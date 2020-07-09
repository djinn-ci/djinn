package main

import (
	"bytes"
	"context"
	"io"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/runner"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type buildRunner struct {
	db        *sqlx.DB
	build     *build.Build
	log       *log.Logger
	block     *crypto.Block
	runner    runner.Runner
	collector runner.Collector
	placer    runner.Placer
	buf       *bytes.Buffer
	bufs      map[int64]*bytes.Buffer
	jobs      map[string]*build.Job
}

func (r *buildRunner) driverJob() *build.Job {
	for _, j := range r.jobs {
		if j.Name == "create driver" {
			return j
		}
	}
	return nil
}

func (r *buildRunner) driverBuffer() *bytes.Buffer {
	for _, j := range r.jobs {
		if j.Name == "create driver" {
			return r.bufs[j.ID]
		}
	}
	return nil
}

func (r *buildRunner) load() error {
	vv, err := build.NewVariableStore(r.db, r.build).All()

	if err != nil {
		return errors.Err(err)
	}

	r.runner.Env = make([]string, 0, len(vv))

	for _, v := range vv {
		r.runner.Env = append(r.runner.Env, v.Key+"="+v.Value)
	}

	kk, err := build.NewKeyStore(r.db, r.build).All()

	if err != nil {
		return errors.Err(err)
	}

	for _, k := range kk {
		r.runner.Objects.Set("key:"+k.Name, k.Location)
	}

	oo, err := build.NewObjectStore(r.db, r.build).All()

	if err != nil {
		return errors.Err(err)
	}

	for _, o := range oo {
		r.runner.Objects.Set(o.Source, o.Name)
	}

	ss, err := build.NewStageStore(r.db, r.build).All(query.OrderAsc("created_at"))

	if err != nil {
		return errors.Err(err)
	}

	stages := make(map[int64]*runner.Stage)

	for _, s := range ss {
		stages[s.ID] = s.Stage()
	}

	jj, err := build.NewJobStore(r.db, r.build).All(query.OrderAsc("created_at"))

	if err != nil {
		return errors.Err(err)
	}

	mm := make([]database.Model, 0, len(jj))

	for _, j := range jj {
		mm = append(mm, j)
	}

	err = build.NewArtifactStore(r.db, r.build).Load(
		"job_id", database.MapKey("id", mm), database.Bind("id", "job_id", mm...),
	)

	if err != nil {
		return errors.Err(err)
	}

	for _, j := range jj {
		stage := stages[j.StageID]

		r.jobs[stage.Name+j.Name] = j
		r.bufs[j.ID] = &bytes.Buffer{}

		job := j.Job(io.MultiWriter(r.buf, r.bufs[j.ID]))

		stage.Add(job)
	}

	for _, s := range ss{
		r.runner.Add(stages[s.ID])
	}

	r.runner.Writer = r.buf
	r.runner.Placer = &placer{
		db:      r.db,
		block:   r.block,
		build:   r.build,
		objects: r.placer,
	}
	r.runner.Collector = build.NewArtifactStoreWithCollector(r.db, r.collector, r.build)
	return nil
}

func (r *buildRunner) run(c context.Context, d runner.Driver) error {
	builds := build.NewStore(r.db)
	jobs := build.NewJobStore(r.db)

	r.runner.HandleDriverCreate(func() {
		j := r.driverJob()

		if err := jobs.Started(j.ID); err != nil {
			r.log.Error.Println("failed to handle driver creation", j.ID, errors.Err(err))
		}
	})

	r.runner.HandleJobStart(func(job runner.Job) {
		if job.Name == "create driver" {
			return
		}

		j := r.jobs[job.Stage+job.Name]

		if err := jobs.Started(j.ID); err != nil {
			r.log.Error.Println("failed to handle job start", j.ID, errors.Err(err))
		}
	})

	r.runner.HandleJobComplete(func(job runner.Job) {
		j := r.jobs[job.Stage+job.Name]

		if err := jobs.Finished(j.ID, r.bufs[j.ID].String(), job.Status); err != nil {
			r.log.Error.Println("failed to handle job finish", j.ID, errors.Err(err))
		}
	})

	if err := builds.Started(r.build.ID); err != nil {
		return errors.Err(err)
	}

	r.runner.Run(c, d)

	if err := builds.Finished(r.build.ID, r.buf.String(), r.runner.Status); err != nil {
		return errors.Err(err)
	}
	return errors.Err(r.updateJobs())
}

func (r *buildRunner) updateJobs() error {
	jobs := build.NewJobStore(r.db, r.build)

	jj, err := jobs.All(query.WhereRaw("finished_at", "IS", "NULL"))

	if err != nil {
		return errors.Err(err)
	}

	for _, j := range jj {
		output := ""

		if buf, ok := r.bufs[j.ID]; ok {
			output = buf.String()
		}

		if err := jobs.Finished(j.ID, output, r.build.Status); err != nil {
			return errors.Err(err)
		}
	}
	return nil
}
