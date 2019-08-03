package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/model/query"
	"github.com/andrewpillar/thrall/runner"
	"github.com/andrewpillar/thrall/server"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"

	"github.com/RichardKnop/machinery/v1"
	qconfig "github.com/RichardKnop/machinery/v1/config"
)

type worker struct {
	server.Server

	db *sqlx.DB

	concurrency int
	driver      string
	timeout     time.Duration

	redisAddr     string
	redisPassword string

	users   *model.UserStore
	builds  *model.BuildStore

	objects   filestore.FileStore
	artifacts filestore.FileStore

	worker *machinery.Worker

	buffers map[int64]*bytes.Buffer
	signals map[int64]chan os.Signal
}

func (w *worker) init() error {
	w.builds = &model.BuildStore{
		DB: w.db,
	}
	w.users = &model.UserStore{
		DB: w.db,
	}

	w.buffers = make(map[int64]*bytes.Buffer)
	w.signals = make(map[int64]chan os.Signal)

	url := "redis://"

	if w.redisPassword != "" {
		url += w.redisPassword + "@"
	}

	url += w.redisAddr

	qcfg := &qconfig.Config{
		Broker:        url,
		DefaultQueue:  "thrall_builds_" + w.driver,
		ResultBackend: url,
	}

	server, err := machinery.NewServer(qcfg)

	if err != nil {
		return errors.Err(err)
	}

	server.RegisterTask("run_build", w.runBuild)

	w.worker = server.NewWorker("thrall-worker-" + w.driver, w.concurrency)

	return nil
}

func (w *worker) serve() error {
	return errors.Err(w.worker.Launch())
}

func (w worker) handleJobStart(b *model.Build, rj runner.Job) {
	s, err := b.StageStore().FindByName(rj.Stage)

	if err != nil || s.IsZero() {
		return
	}

	j, err := s.JobStore().FindByName(rj.Name)

	if err != nil || j.IsZero() {
		return
	}

	j.StartedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	j.Update()
}

func (w worker) handleJobComplete(b *model.Build, rj runner.Job) {
	s, err := b.StageStore().FindByName(rj.Stage)

	if err != nil || s.IsZero() {
		return
	}

	j, err := s.JobStore().FindByName(rj.Name)

	if err != nil || j.IsZero() {
		return
	}

	output := strings.Trim(w.buffers[j.ID].String(), "\n")

	j.Status = rj.Status
	j.Output = sql.NullString{
		String: output,
		Valid:  len(output) > 0,
	}
	j.FinishedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	j.Update()
}

func (w worker) runBuild(id int64) error {
	b, err := w.builds.Find(id)

	if err != nil {
		return errors.Err(err)
	}

	if b.IsZero() {
		return errors.Err(errors.New("build does not exist"))
	}

	if err := b.LoadDriver(); err != nil {
		return errors.Err(err)
	}

	if err := b.LoadObjects(); err != nil {
		return errors.Err(err)
	}

	if err := b.BuildObjectStore().LoadObjects(b.Objects); err != nil {
		return errors.Err(err)
	}

	if err := b.LoadVariables(); err != nil {
		return errors.Err(err)
	}

	if err := b.LoadStages(); err != nil {
		return errors.Err(err)
	}

	if err := b.StageStore().LoadJobs(b.Stages); err != nil {
		return errors.Err(err)
	}

	jobs := make([]*model.Job, 0)

	for _, s := range b.Stages {
		jobs = append(jobs, s.Jobs...)
	}

	if err := b.JobStore().LoadDependencies(jobs); err != nil {
		return errors.Err(err)
	}

	if err := b.JobStore().LoadArtifacts(jobs); err != nil {
		return errors.Err(err)
	}

	objs := runner.NewPassthrough()

	for _, o := range b.Objects {
		objs[o.Source] = o.Name
	}

	env := make([]string, len(b.Variables), len(b.Variables))

	for i, v := range b.Variables {
		env[i] = v.Key + "=" + v.Value
	}

	buf := &bytes.Buffer{}

	r := runner.Runner{
		Writer:    buf,
		Env:       env,
		Objects:   objs,
		Placer:    &database{
			Placer: w.objects,
			build:  b,
			users:  w.users,
		},
		Collector: &database{
			Collector: w.artifacts,
			build:     b,
			users:     w.users,
		},
	}

	createDriverId := int64(0)

	for _, s := range b.Stages {
		rs := &runner.Stage{
			Name:    s.Name,
			CanFail: s.CanFail,
		}

		for _, j := range s.Jobs {
			w.buffers[j.ID] = &bytes.Buffer{}

			if j.Name == "create driver" {
				createDriverId = j.ID
			}

			depends := make([]string, len(j.Dependencies), len(j.Dependencies))

			for i, d := range j.Dependencies {
				depends[i] = d.Name
			}

			artifacts := runner.NewPassthrough()

			for _, a := range j.Artifacts {
				artifacts[a.Source] = a.Hash
			}

			rj := &runner.Job{
				Writer:    io.MultiWriter(buf, w.buffers[j.ID]),
				Name:      j.Name,
				Commands:  strings.Split(j.Commands, "\n"),
				Depends:   depends,
				Artifacts: artifacts,
			}

			rs.Add(rj)
		}

		r.Add(rs)
	}

	dcfg := config.Driver{}

	json.Unmarshal([]byte(b.Driver.Config), &dcfg)

	d, err := driver.NewEnv(io.MultiWriter(buf, w.buffers[createDriverId]), dcfg)

	if err != nil {
		return errors.Err(err)
	}

	b.Status = runner.Running
	b.StartedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := b.Update(); err != nil {
		return errors.Err(err)
	}

	r.HandleJobStart(func(j runner.Job) {
		w.handleJobStart(b, j)
	})

	r.HandleJobComplete(func(j runner.Job) {
		w.handleJobComplete(b, j)
	})

	ctx, cancel := context.WithTimeout(context.Background(), w.timeout)
	defer cancel()

	w.signals[b.ID] = make(chan os.Signal)

	go func() {
		<-w.signals[b.ID]
		cancel()
	}()

	r.Run(ctx, d)

	b.Status = r.Status
	b.Output = sql.NullString{
		String: buf.String(),
		Valid:  true,
	}
	b.FinishedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := b.Update(); err != nil {
		return errors.Err(err)
	}

	jj, err := b.JobStore().All(query.WhereIs("finished_at", "NULL"))

	if err != nil {
		return errors.Err(err)
	}

	for _, j := range jj {
		j.Status = r.Status
		j.Output = sql.NullString{
			String: w.buffers[j.ID].String(),
			Valid:  true,
		}

		if err := j.Update(); err != nil {
			return errors.Err(err)
		}
	}

	return nil
}
