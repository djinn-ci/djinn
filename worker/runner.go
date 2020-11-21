package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/driver"
	"github.com/andrewpillar/djinn/driver/qemu"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/image"
	"github.com/andrewpillar/djinn/log"
	"github.com/andrewpillar/djinn/runner"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Runner is what's used for actually running the build retrieved from the
// worker. This will handle updating the state of the build and its jobs in
// the database whilst the build runs.
type Runner struct {
	initialized bool // if the underlying runner has been fully initialized

	db *sqlx.DB

	block *crypto.Block
	log   *log.Logger

	build *build.Build

	runner    runner.Runner
	placer    runner.Placer
	collector runner.Collector

	drivers *driver.Registry
	config  map[string]map[string]interface{}

	keycfg string            // the .ssh/config to place in to a build
	keys   map[string][]byte // the encrypted private keys to place in to a build

	buf  *bytes.Buffer
	bufs map[int64]*bytes.Buffer

	jobs map[string]*build.Job
}

func (r *Runner) qemuRealpath(b *build.Build, diskdir string) func(string, string) (string, error) {
	return func(arch, name string) (string, error) {
		i, err := image.NewStore(r.db).Get(
			query.Where("user_id", "=", query.Arg(b.UserID)),
			query.Where("name", "=", query.Arg(name)),
		)

		if err != nil {
			return "", errors.Err(err)
		}

		if i.IsZero() {
			name = filepath.Join(strings.Split(name, "/")...)
			return filepath.Join(diskdir, "_base", arch, name), nil
		}
		return filepath.Join(diskdir, i.Hash), nil
	}
}

func (r *Runner) updateJobs(status runner.Status) error {
	jobs := build.NewJobStore(r.db, r.build)

	jj, err := jobs.All(query.Where("finished_at", "IS", query.Lit("NULL")))

	if err != nil {
		return errors.Err(err)
	}

	for _, j := range jj {
		output := ""

		if buf, ok := r.bufs[j.ID]; ok {
			output = buf.String()
		}

		if err := jobs.Finished(j.ID, output, status); err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

// Init initializes the underlying runner.Runner for build execution with
// the build data from the database.
func (r *Runner) Init() error {
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

	keycfg := bytes.NewBufferString(`StrictHostKeyChecking no
UserKnownHostsFile /dev/null
`)

	r.keys = make(map[string][]byte)

	for _, k := range kk {
		r.keys["key:"+k.Name] = k.Key

		keycfg.WriteString(k.Config)
		r.runner.Objects.Set("key:"+k.Name, "/root/.ssh/"+k.Name)
	}

	r.keycfg = keycfg.String()

	if len(kk) > 0 {
		r.runner.Objects.Set("/root/.ssh/config", "/root/.ssh/config")
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

	mm := database.ModelSlice(len(jj), build.JobModel(jj))

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

		stage.Add(j.Job(io.MultiWriter(r.buf, r.bufs[j.ID])))
	}

	for _, s := range ss {
		r.runner.Add(stages[s.ID])
	}

	r.runner.Writer = r.buf
	r.runner.Placer = r.Placer()
	r.runner.Collector = r.Collector()

	r.initialized = true

	return nil
}

// Collector returns the configured runner.Collector implementation to use for
// collecting build artifacts.
func (r *Runner) Collector() runner.Collector {
	return build.NewArtifactStoreWithCollector(r.db, r.collector, r.build)
}

// Placer returns the configured runner.Placer implementation to use for
// placing build objects and keys into a build.
func (r *Runner) Placer() runner.Placer {
	return &placer{
		block:  r.block,
		keycfg: []byte(r.keycfg),
		keys:   r.keys,
		placer: build.NewObjectStoreWithPlacer(r.db, r.placer, r.build),
	}
}

// DriverJob returns the build pseudo-job that was added to the build for
// tracking the progress of driver creation.
func (r *Runner) DriverJob() *build.Job {
	for _, j := range r.jobs {
		if j.Name == "create driver" {
			return j
		}
	}
	return nil
}

// DriverBuffer returns the buffer for the job that created the driver for the
// build's runner. If one cannot be found then nil is returned.
func (r *Runner) DriverBuffer() *bytes.Buffer {
	if j := r.DriverJob(); j != nil {
		if buf, ok := r.bufs[j.ID]; ok {
			return buf
		}
	}
	return nil
}

// Run runs the build with the given driver. This will return the underlying
// status of the runner upon completion, along with any errors that may occur.
// If an underyling error does occur then the returned status will always be
// runner.Failed.
func (r *Runner) Run(ctx context.Context, jobId string, d *build.Driver) (runner.Status, error) {
	if !r.initialized {
		return runner.Failed, errors.New("runner not initialized")
	}

	builds := build.NewStore(r.db)
	jobs := build.NewJobStore(r.db)

	init, err := r.drivers.Get(d.Config["type"])

	if err != nil {
		fmt.Fprintf(r.buf, "driver %s has not been configured for the worker\n", d.Config["type"])
		fmt.Fprintf(r.buf, "killing build...\n")

		if err := builds.Finished(r.build.ID, r.buf.String(), runner.Killed); err != nil {
			r.log.Error.Println(jobId, "failed to mark build as finished", err)
			return runner.Killed, errors.Err(err)
		}

		err := r.updateJobs(runner.Killed)

		r.log.Error.Println(jobId, "failed to update build jobs", err)

		return runner.Killed, errors.Err(err)
	}

	// Merge global driver configuration with manifest driver configuration for
	// driver initialization.
	config := make(map[string]interface{})

	for k, v := range d.Config {
		config[k] = v
	}

	for k, v := range r.config[d.Config["type"]] {
		config[k] = v
	}

	driver := init(io.MultiWriter(r.buf, r.DriverBuffer()), config)

	if q, ok := driver.(*qemu.Driver); ok {
		// If using the qemu driver then make sure we resolve user uploaded
		// images correctly.
		q.Image = strings.Replace(q.Image, "..", "", -1)
		q.Realpath = r.qemuRealpath(r.build, config["disks"].(string))
	}

	r.runner.HandleDriverCreate(func() {
		j := r.DriverJob()

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
		buf := r.bufs[j.ID]

		if err := jobs.Finished(j.ID, buf.String(), job.Status); err != nil {
			r.log.Error.Println("failed to handle job finish", j.ID, errors.Err(err))
		}
	})

	if err := builds.Started(r.build.ID); err != nil {
		r.log.Error.Println(jobId, "failed to mark build as started", err)
		return runner.Failed, errors.Err(err)
	}

	r.runner.Run(ctx, driver)

	if err := builds.Finished(r.build.ID, r.buf.String(), r.runner.Status); err != nil {
		r.log.Error.Println(jobId, "failed to mark build as finished", err)
		return r.runner.Status, errors.Err(err)
	}

	if err := r.updateJobs(r.runner.Status); err != nil {
		r.log.Error.Println(jobId, "failed to update build jobs", err)
		return r.runner.Status, errors.Err(err)
	}
	return r.runner.Status, nil
}
