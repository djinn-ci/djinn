package main

import (
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
	"github.com/andrewpillar/thrall/log"
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
	b, err := w.builds.Get(query.Where("id", "=", id))

	if err != nil {
		return errors.Err(err)
	}

	r := newBuildRunner(w.db, b, w.collector, w.placer)

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

		if err := w.builds.Update(b); err != nil {
			return errors.Err(err)
		}
		return errors.Err(r.updateJobs())
	}

	buildDriver, err := build.NewDriverStore(w.db, b).Get()

	if err != nil {
		return errors.Err(err)
	}

	cfg := make(map[string]string)
	json.Unmarshal([]byte(buildDriver.Config), &cfg)

	driverInit, err := w.drivers.Get(cfg["type"])

	if err != nil {
		fmt.Fprintf(r.buf, "driver %s has not been configured for the worker\n", cfg["type"])
		fmt.Fprintf(r.buf, "killing build...\n")

		b.Status = runner.Killed
		b.Output = sql.NullString{
			String: r.buf.String(),
			Valid:  true,
		}
		b.FinishedAt = pq.NullTime{
			Time:  time.Now(),
			Valid: true,
		}

		if err := w.builds.Update(b); err != nil {
			return errors.Err(err)
		}
		return errors.Err(r.updateJobs())
	}

	if err := r.load(); err != nil {
		return errors.Err(err)
	}

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

	merged := make(map[string]interface{})

	for k, v := range cfg {
		merged[k] = v
	}

	for k, v := range w.driverconf[cfg["type"]] {
		merged[k] = v
	}

	d := driverInit(io.MultiWriter(r.buf, r.driverBuffer()), merged)

	if q, ok := d.(*qemu.QEMU); ok {
		q.Realpath = w.qemuRealPath(b, merged["disks"].(string))
	}
	return errors.Err(r.run(ctx, d))
}
