package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/driver/qemu"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/image"
	"github.com/andrewpillar/thrall/runner"

	"github.com/andrewpillar/query"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"

	"github.com/RichardKnop/machinery/v1"
)

type worker struct {
	db         *sqlx.DB
	redis      *redis.Client
	driverconf map[string]map[string]interface{}
	drivers    *driver.Store
	timeout    time.Duration
	server     *machinery.Server
	worker     *machinery.Worker
	placer     runner.Placer
	collector  runner.Collector
	builds     build.Store
}

func (w *worker) init(name string, concurrency int) {
	w.server.RegisterTask("run_build", w.run)
	w.worker = w.server.NewWorker("thrall-worker-"+name, concurrency)
	w.builds = build.NewStore(w.db)
}

func (w *worker) qemuRealPath(b *build.Build, disks string) func(string, string) (string, error) {
	return func(arch, name string) (string, error) {
		i, err := image.NewStore(w.db).Get(
			query.Where("user_id", "=", b.UserID),
			query.Where("name", "=", name),
		)

		if err != nil {
			return "", err
		}

		if i.IsZero() {
			name = filepath.Join(strings.Split(name, "/")...)
			return filepath.Join(disks, "_base", arch, name), nil
		}
		return filepath.Join(disks, arch, i.Hash), nil
	}
}

func (w *worker) run(id int64) error {
	b := workerBuild{
		db:     w.db,
		builds: w.builds,
	}

	if err := b.load(id); err != nil {
		return errors.Err(err)
	}

	if b.Status == runner.Killed {
		b.Status = runner.Killed
		b.Output = sql.NullString{
			String: "build killed",
			Valid:  true,
		}
		b.FinishedAt = pq.NullTime{
			Time:  time.Now(),
			Valid: true,
		}

		if err := w.builds.Update(b.Build); err != nil {
			return errors.Err(err)
		}
		return errors.Err(b.updateJobs())
	}

	var (
		driverBuffer *bytes.Buffer
		runnerBuffer *bytes.Buffer = &bytes.Buffer{}
	)

	for _, j := range b.jobs {
		if j.Name == "create driver" {
			driverBuffer = b.buffers[j.ID]
			break
		}
	}

	r := runner.Runner{
		Writer:    runnerBuffer,
		Env:       b.vars,
		Objects:   b.objects,
		Placer:    &placer{
			db:      w.db,
			build:   b.Build,
			objects: w.placer,
		},
		Collector: build.NewArtifactStoreWithCollector(w.db, w.collector, b.Build),
	}

	r.Add(b.stages...)

	cfg := make(map[string]string)
	json.Unmarshal([]byte(b.Driver.Config), &cfg)

	b.Status = runner.Running
	b.StartedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := w.builds.Update(b.Build); err != nil {
		return errors.Err(err)
	}

	r.HandleJobStart(b.handleJobStart)
	r.HandleJobComplete(b.handleJobComplete)

	ctx, cancel := context.WithTimeout(context.Background(), w.timeout)
	defer cancel()

	sub := w.redis.Subscribe(fmt.Sprintf("kill-%v", b.ID))
	defer sub.Close()

	go func() {
		msg := <-sub.Channel()

		if msg == nil {
			return
		}
		if msg.Payload == b.Secret.String {
			cancel()
		}
	}()

	driverInit, err := w.drivers.Get(cfg["type"])

	if err != nil {
		fmt.Fprintf(runnerBuffer, "driver %s has not been configured for the worker\n", cfg["type"])
		fmt.Fprintf(runnerBuffer, "killing build...\n")

		b.Status = runner.Killed
		b.Output = sql.NullString{
			String: runnerBuffer.String(),
			Valid:  true,
		}
		b.FinishedAt = pq.NullTime{
			Time:  time.Now(),
			Valid: true,
		}

		if err := w.builds.Update(b.Build); err != nil {
			return errors.Err(err)
		}
		return errors.Err(b.updateJobs())
	}

	merged := make(map[string]interface{})

	for k, v := range cfg {
		merged[k] = v
	}

	for k, v := range w.driverconf[cfg["type"]] {
		merged[k] = v
	}

	d := driverInit(io.MultiWriter(runnerBuffer, driverBuffer), merged)

	if q, ok := d.(*qemu.QEMU); ok {
		q.Realpath = w.qemuRealPath(b.Build, merged["disks"].(string))
	}

	r.Run(ctx, d)

	b.Status = r.Status
	b.Output = sql.NullString{
		String: runnerBuffer.String(),
		Valid:  true,
	}
	b.FinishedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := w.builds.Update(b.Build); err != nil {
		return errors.Err(err)
	}
	return errors.Err(b.updateJobs())
}
