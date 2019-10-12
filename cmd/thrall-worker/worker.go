package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/runner"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/go-redis/redis"

	"github.com/lib/pq"

	"github.com/RichardKnop/machinery/v1"
)

type worker struct {
	client *redis.Client
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
}

func (w *worker) init(qname string) {
	w.builds = model.BuildStore{
		Store: w.store,
	}
	w.users = model.UserStore{
		Store: w.store,
	}

	w.queue.RegisterTask("run_build", w.runBuild)

	w.worker = w.queue.NewWorker("thrall-worker-" + qname, w.concurrency)
}

func (w worker) loadBuild(s string) (*model.Build, error) {
	b := w.builds.New()

	dec := json.NewDecoder(strings.NewReader(s))
	dec.Decode(b)

	if err := b.LoadDriver(); err != nil {
		return b, errors.Err(err)
	}

	if err := b.LoadObjects(); err != nil {
		return b, errors.Err(err)
	}

	if err := b.BuildObjectStore().LoadObjects(b.Objects); err != nil {
		return b, errors.Err(err)
	}

	if err := b.LoadKeys(); err != nil {
		return b, errors.Err(err)
	}

	if err := b.LoadVariables(); err != nil {
		return b, errors.Err(err)
	}

	if err := b.LoadStages(); err != nil {
		return b, errors.Err(err)
	}

	err := b.StageStore().LoadJobs(b.Stages)

	return b, errors.Err(err)
}

func (w worker) runBuild(s string) error {
	b, err := w.loadBuild(s)

	if err != nil {
		return errors.Err(err)
	}

	jj := make([]*model.Job, 0)

	for _, s := range b.Stages {
		jj = append(jj, s.Jobs...)
	}

	jobs := b.JobStore()

	if err := jobs.LoadArtifacts(jj); err != nil {
		return errors.Err(err)
	}

	objs := runner.NewPassthrough()

	for _, o := range b.Objects {
		objs[o.Source] = o.Name
	}

	keyBuffers := make(map[string]*bytes.Buffer)

	sshConfig := &bytes.Buffer{}

	// Use object placement to add keys to the build env. Prefix each name with
	// 'mem:' to tell the placer to look in the database for the key to add.
	for _, k := range b.Keys {
		objs["mem:" + k.Name] = k.Location

		b, err := crypto.Decrypt(k.Key)

		if err != nil {
			continue
		}

		sshConfig.WriteString(k.Config)

		keyBuffers[k.Name] = bytes.NewBuffer(b)
	}

	objs["mem:ssh_config"] = "/root/.ssh/config"
	keyBuffers["ssh_config"] = sshConfig

	env := make([]string, len(b.Variables), len(b.Variables))

	for i, v := range b.Variables {
		env[i] = v.Key + "=" + v.Value
	}

	buffers := make(map[int64]*bytes.Buffer)

	buf := &bytes.Buffer{}

	r := runner.Runner{
		Writer:    buf,
		Env:       env,
		Objects:   objs,
		Placer:    &database{
			Placer:     w.objects,
			memObjects: keyBuffers,
			build:      b,
			users:      w.users,
		},
		Collector: &database{
			Collector:  w.artifacts,
			build:      b,
			users:      w.users,
		},
	}

	mjobs := make(map[string]*model.Job)

	createDriverId := int64(0)

	for _, s := range b.Stages {
		rs := &runner.Stage{
			Name:    s.Name,
			CanFail: s.CanFail,
		}

		for _, j := range s.Jobs {
			buffers[j.ID] = &bytes.Buffer{}
			mjobs[j.Name] = j

			if j.Name == "create driver" {
				createDriverId = j.ID
			}

			artifacts := runner.NewPassthrough()

			for _, a := range j.Artifacts {
				artifacts[a.Source] = a.Hash
			}

			rj := &runner.Job{
				Writer:    io.MultiWriter(buf, buffers[j.ID]),
				Name:      j.Name,
				Commands:  strings.Split(j.Commands, "\n"),
				Artifacts: artifacts,
			}

			rs.Add(rj)
		}

		r.Add(rs)
	}

	cfg := make(map[string]string)

	json.Unmarshal([]byte(b.Driver.Config), &cfg)

	d, err := driver.New(
		io.MultiWriter(buf, buffers[createDriverId]),
		config.Driver{
			Config: cfg,
			SSH:    w.driverCfg.SSH,
			Qemu:   w.driverCfg.Qemu,
		},
	)

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

	r.HandleJobStart(func(rj runner.Job) {
		j := mjobs[rj.Name]

		j.Status = runner.Running
		j.StartedAt = pq.NullTime{
			Time:  time.Now(),
			Valid: true,
		}

		if err := jobs.Update(j); err != nil {
			log.Error.Println(errors.Err(err))
		}
	})

	r.HandleJobComplete(func(rj runner.Job) {
		j := mjobs[rj.Name]

		j.Status = rj.Status
		j.Output = sql.NullString{
			String: buffers[j.ID].String(),
			Valid:  true,
		}
		j.FinishedAt = pq.NullTime{
			Time:  time.Now(),
			Valid: true,
		}

		if err := jobs.Update(j); err != nil {
			log.Error.Println(errors.Err(err))
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), w.timeout)
	defer cancel()

	sub := w.client.Subscribe("kill")
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
			String: buffers[j.ID].String(),
			Valid:  true,
		}
		j.FinishedAt = pq.NullTime{
			Time:  time.Now(),
			Valid: true,
		}

		if err := jobs.Update(j); err != nil {
			return errors.Err(err)
		}
	}

	return nil
}
