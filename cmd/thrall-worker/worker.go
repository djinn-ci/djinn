package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/collector"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/placer"
	"github.com/andrewpillar/thrall/runner"
	"github.com/andrewpillar/thrall/server"

	"github.com/jmoiron/sqlx"

	"github.com/RichardKnop/machinery/v1"
	qconfig "github.com/RichardKnop/machinery/v1/config"
)

type worker struct {
	server.Server

	db *sqlx.DB

	concurrency int
	driver      string

	redisAddr     string
	redisPassword string

	users   *model.UserStore
	builds  *model.BuildStore

	placer    runner.Placer
	collector runner.Collector

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

func (w worker) runBuild(id int64) error {
	b, err := w.builds.Find(id)

	if err != nil {
		return errors.Err(err)
	}

	if b.IsZero() {
		return errors.Err(errors.New("build does not exist"))
	}

	if err := b.LoadUser(); err != nil {
		return errors.Err(err)
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

	jobs := make(map[int64]*model.Job)

	for _, s := range b.Stages {
		for _, j := range s.Jobs {
			if !j.ParentID.Valid {
				jobs[j.ID] = j
				continue
			}

			if parent, ok := jobs[j.ParentID.Int64]; ok {
				parent.Dependencies = append(parent.Dependencies, j)
			}
		}

		s.Jobs = make([]*model.Job, 0)
	}

	for _, s := range b.Stages {
		for _, j := range jobs {
			if j.StageID == s.ID {
				s.Jobs = append(s.Jobs, j)
			}
		}
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

	pl := placer.NewDatabase(w.placer)
	cl := collector.NewDatabase(w.collector)

	pl.Build = b
	pl.Users = w.users

	cl.Build = b

	w.signals[b.ID] = make(chan os.Signal)

	r := runner.NewRunner(buf, env, objs, pl, cl, w.signals[b.ID])

	createDriverId := int64(0)

	for _, s := range b.Stages {
		rs := runner.NewStage(s.Name, s.CanFail)

		if err := s.JobStore().LoadDependencies(s.Jobs); err != nil {
			return errors.Err(err)
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

			rj := runner.NewJob(
				io.MultiWriter(buf, w.buffers[j.ID]),
				j.Name,
				strings.Split(j.Commands, "\n"),
				depends,
				runner.NewPassthrough(),
			)

			rs.Add(rj)
		}

		r.Add(rs)
	}

	dcfg := config.Driver{}

	json.Unmarshal([]byte(b.Driver.Config), &dcfg)

	d, err := driver.NewEnv(w.buffers[createDriverId], dcfg)

	if err != nil {
		return errors.Err(err)
	}

	r.Run(d)

	b.Status = r.Status
	b.Output = sql.NullString{
		String: buf.String(),
		Valid:  true,
	}

	if err := b.Update(); err != nil {
		return errors.Err(err)
	}

	return nil
}
