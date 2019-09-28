package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	nethttp "net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/http"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/runner"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/lib/pq"

	"github.com/RichardKnop/machinery/v1"
)

type worker struct {
	*http.Server

	queue  *machinery.Server
	router *mux.Router

	store model.Store

	concurrency int
	driverCfg   config.Driver
	timeout     time.Duration

	users  model.UserStore
	builds model.BuildStore

	objects   filestore.FileStore
	artifacts filestore.FileStore

	worker *machinery.Worker

	buffers map[int64]*bytes.Buffer
	signals map[int64]chan struct{}
}

func (wrk worker) killBuild(builds model.BuildStore, users model.UserStore) nethttp.HandlerFunc {
	resp := make(map[string]string)

	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		vars := mux.Vars(r)

		id, _ := strconv.ParseInt(vars["build"], 10, 64)

		b, err := builds.Find(id)

		if err != nil {
			resp["message"] = errors.Cause(err).Error()

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(nethttp.StatusInternalServerError)

			enc := json.NewEncoder(w)
			enc.Encode(&resp)
			return
		}

		body := make(map[string]string)

		dec := json.NewDecoder(r.Body)
		dec.Decode(&body)

		secret, ok := body["secret"]

		if !ok {
			w.WriteHeader(nethttp.StatusNotFound)
			return
		}

		if b.KillSecret.String != secret {
			w.WriteHeader(nethttp.StatusNotFound)
			return
		}

		ch, ok := wrk.signals[id]

		if !ok {
			w.WriteHeader(nethttp.StatusNotFound)
			return
		}

		ch <- struct{}{}

		w.WriteHeader(nethttp.StatusNoContent)
	})
}

func (w *worker) init(qname string) {
	w.builds = model.BuildStore{
		Store: w.store,
	}
	w.users = model.UserStore{
		Store: w.store,
	}

	w.buffers = make(map[int64]*bytes.Buffer)
	w.signals = make(map[int64]chan struct{})

	w.queue.RegisterTask("run_build", w.runBuild)

	w.worker = w.queue.NewWorker("thrall-worker-" + qname, w.concurrency)

	w.router = mux.NewRouter()

	w.router.HandleFunc("/kill/{build:[0-9]+}", w.killBuild(w.builds, w.users)).Methods("DELETE")

	w.Server.Init(w.router)
}

func (w worker) handleJobStart(b *model.Build, rj runner.Job) {
	s, err := b.StageStore().FindByName(rj.Stage)

	if err != nil || s.IsZero() {
		return
	}

	jobs := s.JobStore()

	j, err := jobs.FindByName(rj.Name)

	if err != nil || j.IsZero() {
		return
	}

	j.StartedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	jobs.Update(j)
}

func (w worker) handleJobComplete(b *model.Build, rj runner.Job) {
	s, err := b.StageStore().FindByName(rj.Stage)

	if err != nil || s.IsZero() {
		return
	}

	jobs := s.JobStore()

	j, err := jobs.FindByName(rj.Name)

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

	jobs.Update(j)
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

	jj := make([]*model.Job, 0)

	for _, s := range b.Stages {
		jj = append(jj, s.Jobs...)
	}

	jobs := b.JobStore()

	if err := jobs.LoadDependencies(jj); err != nil {
		return errors.Err(err)
	}

	if err := jobs.LoadArtifacts(jj); err != nil {
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

	cfg := make(map[string]string)

	json.Unmarshal([]byte(b.Driver.Config), &cfg)

	d, err := driver.New(
		io.MultiWriter(buf, w.buffers[createDriverId]),
		config.Driver{
			Config: cfg,
			SSH:    w.driverCfg.SSH,
			Qemu:   w.driverCfg.Qemu,
		},
	)

	if err != nil {
		return errors.Err(err)
	}

	b.KillAddr = sql.NullString{
		String: w.Server.Addr,
		Valid:  true,
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
		w.handleJobStart(b, j)
	})

	r.HandleJobComplete(func(j runner.Job) {
		w.handleJobComplete(b, j)
	})

	ctx, cancel := context.WithTimeout(context.Background(), w.timeout)
	defer cancel()

	w.signals[b.ID] = make(chan struct{})
	done := make(chan struct{})

	go func() {
		r.Run(ctx, d)
		done <- struct{}{}
	}()


	select {
	case <-w.signals[b.ID]:
		r.Status = runner.Killed
		cancel()
	case <-done:
		break
	}

	b.KillAddr = sql.NullString{}
	b.Status = r.Status
	b.Output = sql.NullString{
		String: buf.String(),
		Valid:  true,
	}
	b.FinishedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := w.builds.Update(b); err != nil {
		return errors.Err(err)
	}

	jj, err = jobs.All(query.WhereRaw("finished_at", "IS", "NULL"))

	if err != nil {
		return errors.Err(err)
	}

	for _, j := range jj {
		j.Status = r.Status
		j.Output = sql.NullString{
			String: w.buffers[j.ID].String(),
			Valid:  true,
		}

		if err := jobs.Update(j); err != nil {
			return errors.Err(err)
		}
	}

	return nil
}
