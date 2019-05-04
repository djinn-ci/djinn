package main

import (
	"bytes"
	"os"

	thrallconfig "github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/collector"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/placer"
	"github.com/andrewpillar/thrall/runner"
	"github.com/andrewpillar/thrall/server"

	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
)

type worker struct {
	server.Server

	store  *model.Store

	concurrency int

	redisAddr     string
	redisPassword string

	users   *model.UserStore
	objects *model.ObjectStore
	builds  *model.BuildStore

	placer    runner.Placer
	collector runner.Collector

	worker *machinery.Worker

	signals map[int64]chan os.Signal
}

func (w *worker) init() error {
	w.users = &model.UserStore{
		Store: w.store,
	}
	w.objects = &model.ObjectStore{
		Store: w.store,
	}
	w.builds = &model.BuildStore{
		Store: w.store,
	}
	w.signals = make(map[int64]chan os.Signal)

	url := "redis://"

	if w.redisPassword != "" {
		url += w.redisPassword + "@"
	}

	url += w.redisAddr

	cnf := &config.Config{
		Broker:        url,
		DefaultQueue:  "thrall_builds",
		ResultBackend: url,
	}

	server, err := machinery.NewServer(cnf)

	if err != nil {
		return errors.Err(err)
	}

	server.RegisterTask("run_build", w.runBuild)

	w.worker = server.NewWorker("thrall_builds", w.concurrency)

	return nil
}

func (w *worker) serve() error {
	return errors.Err(w.worker.Launch())
}

func (w worker) runBuild(id int64) {
	log.Debug.Println("received build", id)

	b, err := w.builds.Find(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		return
	}

	w.signals[id] = make(chan os.Signal)

	buf := &bytes.Buffer{}

	placer := placer.NewDatabase(w.placer)
	collector := collector.NewDatabase(w.collector)

	placer.Build = b
	placer.Users = w.users

	collector.Build = b

	if err := b.LoadVariables(); err != nil {
		log.Error.Println(errors.Err(err))
		return
	}

	env := make([]string, len(b.Variables), len(b.Variables))

	for i, v := range b.Variables {
		env[i] = v.Variable.Key + "=" + v.Variable.Value
	}

	r := runner.NewRunner(
		buf,
		env,
		[]thrallconfig.Passthrough{},
		placer,
		collector,
		w.signals[id],
	)

	ss, err := b.StageStore().All()

	if err != nil {
		log.Error.Println(errors.Err(err))
		return
	}

	for _, s := range ss {
		r.Add(s.Stage())
	}

	jobs := b.JobStore()

	jj, err := jobs.All()

	if err != nil {
		log.Error.Println(errors.Err(err))
		return
	}

	if err := jobs.LoadRelations(jj); err != nil {
		log.Error.Println(errors.Err(err))
		return
	}

	for _, s := range r.Stages {
		for _, j := range jj {
			if s.Name != j.Stage.Name {
				continue
			}

			s.Add(j.Job())
		}
	}
}
