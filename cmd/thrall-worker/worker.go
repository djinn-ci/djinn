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
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/driver/ssh"
	"github.com/andrewpillar/thrall/driver/qemu"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/image"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/runner"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"

	"github.com/RichardKnop/machinery/v1"
)

type worker struct {
	db        *sqlx.DB
	redis     *redis.Client
	driver    config.Driver
	timeout   time.Duration
	server    *machinery.Server
	worker    *machinery.Worker
	placer    runner.Placer
	collector runner.Collector
	users     user.Store
	builds    build.Store
	jobs      build.JobStore
}

func (w *worker) init(name string, concurrency int) {
	w.server.RegisterTask("run_build", w.run)
	w.worker = w.server.NewWorker("thrall-worker-"+name, concurrency)
	w.users = user.NewStore(w.db)
	w.builds = build.NewStore(w.db)
	w.jobs = build.NewJobStore(w.db)
}

func (w *worker) qemuRealPath(b *build.Build) func(string, string) (string, error) {
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
			return filepath.Join(w.driver.QEMU.Disks, "_base", arch, name), nil
		}
		return filepath.Join(w.driver.QEMU.Disks, arch, i.Hash), nil
	}
}

func (w *worker) configureDrivers(b *build.Build, cfg map[string]string) {
	runner.ConfigureDriver(
		"qemu",
		qemu.Configure(
			qemu.Key(w.driver.QEMU.Key),
			qemu.CPUs(w.driver.QEMU.CPUs),
			qemu.Memory(w.driver.QEMU.Memory),
			qemu.Image(cfg["image"]),
			qemu.Realpath(w.qemuRealPath(b)),
		),
	)

	runner.ConfigureDriver(
		"ssh",
		ssh.Configure(
			ssh.User(w.driver.SSH.User),
			ssh.Key(w.driver.SSH.Key),
			ssh.Timeout(time.Duration(time.Second*time.Duration(w.driver.SSH.Timeout))),
		),
	)
}

func (w *worker) getBuildObjects(b *build.Build) (runner.Passthrough, error) {
	objs := runner.Passthrough{}

	kk, err := build.NewKeyStore(w.db, b).All()

	if err != nil {
		return objs, errors.Err(err)
	}

	for _, k := range kk {
		objs.Set("key:"+k.Name, k.Location)
	}

	oo, err := build.NewObjectStore(w.db, b).All()

	if err != nil {
		return objs, errors.Err(err)
	}

	for _, o := range oo {
		objs.Set(o.Source, o.Name)
	}
	return objs, nil
}

func (w *worker) getBuildVars(b *build.Build) ([]string, error) {
	env := make([]string, 0)

	vv, err := build.NewVariableStore(w.db, b).All()

	if err != nil {
		return env, errors.Err(err)
	}

	for _, v := range vv {
		env = append(env, v.Key+"="+v.Value)
	}
	return env, nil
}

func (w *worker) getBuildStages(b *build.Build) (map[int64]*runner.Stage, error) {
	m := make(map[int64]*runner.Stage)
	ss, err := build.NewStageStore(w.db, b).All()

	if err != nil {
		return m, errors.Err(err)
	}

	for _, s := range ss {
		m[s.ID] = &runner.Stage{
			Name:    s.Name,
			CanFail: s.CanFail,
		}
	}
	return m, nil
}

func (w *worker) getBuildJobs(b *build.Build, stages map[int64]*runner.Stage) (map[int64]*runner.Job, error) {
	m := make(map[int64]*runner.Job)
	jj, err := build.NewJobStore(w.db, b).All()

	if err != nil {
		return m, errors.Err(err)
	}

	aa, err := build.NewArtifactStore(w.db, b).All()

	if err != nil {
		return m, errors.Err(err)
	}

	for _, j := range jj {
		m[j.ID] = &runner.Job{
			Name:     m[j.StageID].Name,
			Commands: strings.Split(j.Commands, "\n"),
		}
		stages[j.StageID].Add(m[j.ID])
	}

	for _, a := range aa {
		m[a.JobID].Artifacts.Set(a.Source, a.Hash)
	}
	return m, nil
}

func (w *worker) run(s string) error {
	b := w.builds.New()

	json.NewDecoder(strings.NewReader(s)).Decode(b)

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
		return errors.Err(w.updateJobs(b, make(map[int64]*bytes.Buffer)))
	}

	objs, err := w.getBuildObjects(b)

	if err != nil {
		return errors.Err(err)
	}

	vars, err := w.getBuildVars(b)

	if err != nil {
		return errors.Err(err)
	}

	stages, err := w.getBuildStages(b)

	if err != nil {
		return errors.Err(err)
	}

	jobs, err := w.getBuildJobs(b, stages)

	if err != nil {
		return errors.Err(err)
	}

	buildDriver, err := build.NewDriverStore(w.db, b).Get()

	if err != nil {
		return errors.Err(err)
	}

	var (
		driverBuffer *bytes.Buffer
		runnerBuffer *bytes.Buffer = &bytes.Buffer{}
	)

	jobBuffers := make(map[int64]*bytes.Buffer)
	jobIds := make(map[string]int64)

	r := runner.Runner{
		Writer:    runnerBuffer,
		Env:       vars,
		Objects:   objs,
		Placer:    &placer{
			db:      w.db,
			build:   b,
			objects: w.placer,
		},
		Collector: build.NewArtifactStoreWithCollector(w.db, w.collector, b),
	}

	for id, job := range jobs {
		jobBuffers[id] = &bytes.Buffer{}
		jobIds[job.Name] = id

		job.Writer = io.MultiWriter(runnerBuffer, jobBuffers[id])

		if job.Name == "create driver" {
			driverBuffer = jobBuffers[id]
		}
	}

	for _, stage := range stages {
		r.Add(stage)
	}

	cfg := make(map[string]string)
	json.Unmarshal([]byte(buildDriver.Config), &cfg)

	w.configureDrivers(b, cfg)

	configure, err := runner.GetDriver(cfg["type"])

	if err != nil {
		return errors.Err(err)
	}

	d, err := configure(io.MultiWriter(runnerBuffer, driverBuffer))

	if err != nil {
		return errors.Err(err)
	}

	b.Status = runner.Running
	b.StartedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := w.builds.Update(b); err != nil {
		return errors.Err(err)
	}

	r.HandleJobStart(func(j runner.Job) {
		id := jobIds[j.Name]

		err := w.jobs.Update(&build.Job{
			ID:        id,
			Status:    j.Status,
			StartedAt: pq.NullTime{
				Time:  time.Now(),
				Valid: true,
			},
		})

		if err != nil {
			log.Error.Println(errors.Err(err))
		}
	})

	r.HandleJobComplete(func(j runner.Job) {
		id := jobIds[j.Name]

		err := w.jobs.Update(&build.Job{
			ID:     id,
			Status: j.Status,
			Output: sql.NullString{
				String: jobBuffers[id].String(),
				Valid:  true,
			},
			FinishedAt: pq.NullTime{
				Time:  time.Now(),
				Valid: true,
			},
		})

		if err != nil {
			log.Error.Println(errors.Err(err))
		}
	})

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

	if err := w.builds.Update(b); err != nil {
		return errors.Err(err)
	}
	return errors.Err(w.updateJobs(b, jobBuffers))
}

func (w *worker) updateJobs(b *build.Build, buffers map[int64]*bytes.Buffer) error {
	jobs := build.NewJobStore(w.db, b)

	jj, err := jobs.All(query.WhereRaw("finished_at", "IS", "NULL"))

	if err != nil {
		return errors.Err(err)
	}

	for _, j := range jj {
		j.Status = b.Status

		if buf, ok := buffers[j.ID]; ok {
			j.Output = sql.NullString{
				String: buf.String(),
				Valid:  true,
			}
		}

		j.FinishedAt = pq.NullTime{
			Time:  time.Now(),
			Valid: true,
		}
	}

	err = jobs.Update(jj...)
	return errors.Err(err)
}
