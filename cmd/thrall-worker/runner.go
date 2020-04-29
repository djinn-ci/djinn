package main

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"time"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/runner"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type buildRunner struct {
	db        *sqlx.DB
	build     *build.Build
	runner    runner.Runner
	collector runner.Collector
	placer    runner.Placer
	buf       *bytes.Buffer
	bufs      map[int64]*bytes.Buffer
	jobs      map[string]*build.Job
}

func newBuildRunner(db *sqlx.DB, b *build.Build, c runner.Collector, p runner.Placer) *buildRunner {
	return &buildRunner{
		db:        db,
		build:     b,
		collector: c,
		placer:    p,
		buf:       &bytes.Buffer{},
		bufs:      make(map[int64]*bytes.Buffer),
		jobs:      make(map[string]*build.Job),
	}
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

	mm := make([]model.Model, 0, len(jj))

	for _, j := range jj {
		mm = append(mm, j)
	}

	err = build.NewArtifactStore(r.db, r.build).Load(
		"job_id", model.MapKey("id", mm), model.Bind("id", "job_id", mm...),
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

	for _, stage := range stages {
		r.runner.Add(stage)
	}

	r.runner.Writer = r.buf
	r.runner.Placer = &placer{
		db:      r.db,
		build:   r.build,
		objects: r.placer,
	}
	r.runner.Collector = build.NewArtifactStoreWithCollector(r.db, r.collector, r.build)
	return nil
}

func (r *buildRunner) run(c context.Context, d runner.Driver) error {
	builds := build.NewStore(r.db)

	r.runner.HandleJobStart(func(job runner.Job) {
		j := r.jobs[job.Stage+job.Name]
		j.Status = job.Status
		j.StartedAt = pq.NullTime{
			Time:  time.Now(),
			Valid: true,
		}

		if err := build.NewJobStore(r.db).Update(j); err != nil {
			log.Error.Println(errors.Err(err))
		}
	})

	r.runner.HandleJobComplete(func(job runner.Job) {
		j := r.jobs[job.Stage+job.Name]
		j.Status = job.Status
		j.Output = sql.NullString{
			String: r.bufs[j.ID].String(),
			Valid:  true,
		}
		j.FinishedAt = pq.NullTime{
			Time:  time.Now(),
			Valid: true,
		}

		if err := build.NewJobStore(r.db).Update(j); err != nil {
			log.Error.Println(errors.Err(err))
		}
	})

	r.build.Status = runner.Running
	r.build.StartedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := builds.Update(r.build); err != nil {
		return errors.Err(err)
	}

	r.runner.Run(c, d)

	r.build.Status = r.runner.Status
	r.build.Output = sql.NullString{
		String: r.buf.String(),
		Valid:  true,
	}
	r.build.FinishedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := builds.Update(r.build); err != nil {
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
		j.Status = r.build.Status

		if buf, ok := r.bufs[j.ID]; ok {
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
	return errors.Err(jobs.Update(jj...))
}
