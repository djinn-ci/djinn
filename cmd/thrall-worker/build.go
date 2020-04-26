package main

import (
	"bytes"
	"database/sql"
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

type workerBuild struct {
	*build.Build

	db      *sqlx.DB
	builds  build.Store
	objects runner.Passthrough
	jobs    map[string]*build.Job
	buffers map[int64]*bytes.Buffer
	vars    []string
	stages  []*runner.Stage

	Driver  *build.Driver
}

func (b *workerBuild) load(id int64) error {
	var err error

	b.jobs = make(map[string]*build.Job)
	b.buffers = make(map[int64]*bytes.Buffer)

	b.Build, err = b.builds.Get(query.Where("id", "=", id))

	if err != nil {
		return errors.Err(err)
	}

	b.Driver, err = build.NewDriverStore(b.db, b).Get()

	if err != nil {
		return errors.Err(err)
	}

	vv, err := build.NewVariableStore(b.db, b.Build).All()

	if err != nil {
		return errors.Err(err)
	}

	b.vars = make([]string, 0, len(vv))

	for _, v := range vv {
		b.vars = append(b.vars, v.Key+"="+v.Value)
	}

	kk, err := build.NewKeyStore(b.db, b.Build).All()

	if err != nil {
		return errors.Err(err)
	}

	for _, k := range kk {
		b.objects.Set("key:"+k.Name, k.Location)
	}

	oo, err := build.NewObjectStore(b.db, b.Build).All()

	if err != nil {
		return errors.Err(err)
	}

	for _, o := range oo {
		b.objects.Set(o.Source, o.Name)
	}

	ss, err := build.NewStageStore(b.db, b.Build).All()

	if err != nil {
		return errors.Err(err)
	}

	stages := make(map[int64]*runner.Stage)

	for _, s := range ss {
		stages[s.ID] = s.Stage()
	}

	jj, err := build.NewJobStore(b.db, b.Build).All()

	if err != nil {
		return errors.Err(err)
	}

	mm := make([]model.Model, 0, len(jj))

	for _, j := range jj {
		mm = append(mm, j)
	}

	err = build.NewArtifactStore(b.db, b.Build).Load(
		"job_id", model.MapKey("id", mm), model.Bind("id", "job_id", mm...),
	)

	if err != nil {
		return errors.Err(err)
	}

	for _, j := range jj {
		b.buffers[j.ID] = &bytes.Buffer{}

		stage := stages[j.StageID]
		job := j.Job(b.buffers[j.ID])

		b.jobs[stage.Name+job.Name] = j

		stages[j.StageID].Add(job)
	}
	return nil
}

func (b *workerBuild) handleJobStart(j runner.Job) {
	m := b.jobs[j.Stage+j.Name]
	m.Status = j.Status
	m.StartedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := build.NewJobStore(b.db).Update(m); err != nil {
		log.Error.Println(errors.Err(err))
	}
}

func (b *workerBuild) handleJobComplete(j runner.Job) {
	m := b.jobs[j.Stage+j.Name]
	m.Status = j.Status
	m.Output = sql.NullString{
		String: b.buffers[m.ID].String(),
		Valid:  true,
	}
	m.FinishedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := build.NewJobStore(b.db).Update(m); err != nil {
		log.Error.Println(errors.Err(err))
	}
}

func (b *workerBuild) updateJobs() error {
	jobs := build.NewJobStore(b.db, b.Build)

	jj, err := jobs.All(query.WhereRaw("finished_at", "IS", "NULL"))

	if err != nil {
		return errors.Err(err)
	}

	for _, j := range jj {
		j.Status = b.Build.Status

		if buf, ok := b.buffers[j.ID]; ok {
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
