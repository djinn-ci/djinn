package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"djinn-ci.com/build"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/driver/qemu"
	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/log"
	"djinn-ci.com/runner"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/query"

	"golang.org/x/text/transform"
)

type runnerJob struct {
	id  int64
	buf *bytes.Buffer

	// flushes underlying transform.Writer used for masking variables.
	flush    func() error
	finished bool
}

type Runner struct {
	runner.Runner

	initialized bool

	db     database.Pool
	aesgcm *crypto.AESGCM
	log    *log.Logger
	buf    *bytes.Buffer

	build *build.Build

	builds *build.Store
	images *image.Store
	vars   build.VariableStore
	keys   build.KeyStore
	stages build.StageStore
	jobs   build.JobStore

	objects   *build.ObjectStore
	artifacts *build.ArtifactStore

	masker transform.Transformer

	driver     string
	driverinit driver.Init
	drivercfg  driver.Config

	runnerJobs map[string]runnerJob
	driverJob  *runnerJob
}

// loadAndAddJobs get's all of the jobs for the runner's underlying build and
// adds them to the respective stage in the given map. The jobs that are loaded
// are stored in the runnerJobs map using the stage name and job name as the
// key.
func (r *Runner) loadAndAddJobs(stages map[int64]*runner.Stage) error {
	jj, err := r.jobs.All(
		query.Where("build_id", "=", query.Arg(r.build.ID)),
		query.OrderAsc("created_at"),
	)

	if err != nil {
		return errors.Err(err)
	}

	mm := make([]database.Model, 0, len(jj))

	for _, j := range jj {
		mm = append(mm, j)
	}

	if err := r.artifacts.Load("id", "job_id", mm...); err != nil {
		return errors.Err(err)
	}

	if r.runnerJobs == nil {
		r.runnerJobs = make(map[string]runnerJob)
	}

	for _, j := range jj {
		artifacts := runner.Passthrough{}

		for _, a := range j.Artifacts {
			artifacts.Set(a.Source, a.Name)
		}

		var buf bytes.Buffer

		wc := transform.NewWriter(io.MultiWriter(r.buf, &buf), r.masker)

		job := runnerJob{
			id:    j.ID,
			buf:   &buf,
			flush: wc.Close,
		}

		stage := stages[j.StageID]

		if j.Name == "create-driver" && strings.HasPrefix(stage.Name, "setup - #") {
			r.driverJob = &job
		}

		r.runnerJobs[stage.Name+j.Name] = job

		stage.Add(&runner.Job{
			Writer:    wc,
			Name:      j.Name,
			Commands:  strings.Split(j.Commands, "\n"),
			Artifacts: artifacts,
		})
	}
	return nil
}

// qemuRealpath returns the RealpathFunc to use for resolving the path to the
// image to use for the given build. If the given build is in a namespace then
// the search for the image will be performed within that namespace. Otherwise
// the search will be done for the owner of the build.
//
// If an image could not be found for either the owner of the build or the
// namespace, then the path to the QEMU base images will be returned.
func (r *Runner) qemuRealpath(b *build.Build, diskdir string) qemu.RealpathFunc {
	return func(arch, name string) (string, error) {
		col := "user_id"
		arg := query.Arg(b.UserID)

		// Build was submitted to a namespace so only use custom images in the
		// build's namespace.
		if b.NamespaceID.Valid {
			col = "namespace_id"
			arg = query.Arg(b.NamespaceID)
		}

		i, ok, err := r.images.Get(
			query.Where(col, "=", arg),
			query.Where("name", "=", query.Arg(name)),
		)

		if err != nil {
			return "", errors.Err(err)
		}

		if !ok {
			name = filepath.Join(strings.Split(name, "/")...)
			return filepath.Join(diskdir, "_base", "qemu", arch, name), nil
		}
		return filepath.Join(diskdir, strconv.FormatInt(i.UserID, 10), "qemu", i.Hash), nil
	}
}

// envvars turns the given slice of build variables into a slice of environment
// variables in the format of key=value.
func envvars(vv []*build.Variable) []string {
	env := make([]string, 0, len(vv))

	for _, v := range vv {
		env = append(env, v.Key+"="+v.Value)
	}
	return env
}

// runnerStages gets all of the stages for the underlying build being run,
// and returns them as a map of stages that can be added to the underlying
// Runner. The stages in the map will be under their respective ID from the
// database.
func (r *Runner) runnerStages() (map[int64]*runner.Stage, []int64, error) {
	ss, err := r.stages.All(
		query.Where("build_id", "=", query.Arg(r.build.ID)),
		query.OrderAsc("created_at"),
	)

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	stages := make(map[int64]*runner.Stage)
	order := make([]int64, 0, len(ss))

	for _, s := range ss {
		stages[s.ID] = &runner.Stage{
			Name:    s.Name,
			CanFail: s.CanFail,
		}
		order = append(order, s.ID)
	}
	return stages, order, nil
}

var (
	ErrInitialized    = errors.New("worker: runner initialized")
	ErrNotInitialized = errors.New("worker: runner not initialized")
)

// Init initializes the underlying Runner for running the build. If the runner
// has already been initialized then ErrInitialized is returned.
func (r *Runner) Init() error {
	if r.initialized {
		return ErrInitialized
	}

	vv, err := r.vars.All(query.Where("build_id", "=", query.Arg(r.build.ID)))

	if err != nil {
		return errors.Err(err)
	}

	maskChain := make([]transform.Transformer, 0, len(vv))

	for _, v := range vv {
		if err := variable.Unmask(r.aesgcm, v.Variable); err != nil {
			return errors.Err(err)
		}

		if v.Masked {
			maskChain = append(maskChain, variable.Masker(v.Value))
		}
	}

	r.masker = transform.Chain(maskChain...)

	r.Runner.Env = envvars(vv)

	kk, err := r.keys.All(query.Where("build_id", "=", query.Arg(r.build.ID)))

	if err != nil {
		return errors.Err(err)
	}

	if len(kk) > 0 {
		for _, k := range kk {
			r.Runner.Objects.Set("key:"+k.Name, "/root/.ssh/"+k.Name)
		}
		r.Runner.Objects.Set("/root/.ssh/config", "/root/.ssh/config")
	}

	oo, err := r.objects.All(query.Where("build_id", "=", query.Arg(r.build.ID)))

	if err != nil {
		return errors.Err(err)
	}

	for _, o := range oo {
		r.Runner.Objects.Set(o.Source, o.Name)
	}

	stages, order, err := r.runnerStages()

	if err != nil {
		return errors.Err(err)
	}

	if err := r.loadAndAddJobs(stages); err != nil {
		return errors.Err(err)
	}

	for _, id := range order {
		r.Runner.Add(stages[id])
	}

	r.Runner.Writer = r.buf
	r.Runner.Placer = r.objects.Placer(r.build, r.aesgcm, kk)
	r.Runner.Collector = r.artifacts.Collector(r.build)

	r.initialized = true
	return nil
}

// Tail returns the last 15 lines of what was written to the runner's buffer.
func (r *Runner) Tail() string {
	parts := strings.Split(r.buf.String(), "\n")

	if len(parts) >= 15 {
		parts = parts[len(parts)-15:]
	}
	return strings.Join(parts, "\n")
}

func (r *Runner) updateUnfinishedJobs(ctx context.Context, status runner.Status) error {
	tx, err := r.db.Begin(ctx)

	if err != nil {
		return errors.Err(err)
	}

	defer tx.Rollback(ctx)

	for _, job := range r.runnerJobs {
		if job.finished {
			continue
		}

		if err := r.jobs.FinishedTx(ctx, tx, job.id, job.buf.String(), status); err != nil {
			return errors.Err(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (r *Runner) Run(ctx context.Context, jobId string, d *build.Driver) (runner.Status, error) {
	if !r.initialized {
		return runner.Failed, ErrNotInitialized
	}

	if r.build.StartedAt.Valid {
		if err := r.builds.Orphan(r.build.ID, r.build.UserID); err != nil {
			r.log.Error.Println(jobId, "failed to orphan build", err)
			return runner.Failed, errors.Err(err)
		}
	}

	typ := d.Config["type"]

	if typ != r.driver {
		fmt.Fprintf(r.buf, "driver %s has not been configured for the worker\n", typ)
		fmt.Fprintf(r.buf, "killing build...\n")

		if err := r.builds.Finished(r.build.ID, r.buf.String(), runner.Killed); err != nil {
			r.log.Error.Println(jobId, "failed to mark build as finished", err)
			return runner.Killed, errors.Err(err)
		}

		if err := r.updateUnfinishedJobs(ctx, runner.Killed); err != nil {
			r.log.Error.Println(jobId, "failed to update build jobs", err)
		}
		return runner.Killed, nil
	}

	cfg := r.drivercfg.Merge(d.Config)

	driver := r.driverinit(io.MultiWriter(r.buf, r.driverJob.buf), cfg)

	if q, ok := driver.(*qemu.Driver); ok {
		qemucfg := cfg.(*qemu.Config)

		// If using the qemu driver then make sure we resolve user uploaded
		// images correctly, and that we sanitize the image name.
		q.Image = strings.Replace(q.Image, "..", "", -1)
		q.Realpath = r.qemuRealpath(r.build, qemucfg.Disks)
	}

	r.Runner.HandleDriverCreate(func() {
		if err := r.jobs.Started(r.driverJob.id); err != nil {
			r.log.Error.Println("failed to handle driver creation", jobId, errors.Err(err))
		}
	})

	r.Runner.HandleJobStart(func(job runner.Job) {
		if job.Name == "create-driver" {
			return
		}

		j := r.runnerJobs[job.Stage+job.Name]

		if err := r.jobs.Started(j.id); err != nil {
			r.log.Error.Println("failed to handle job start", jobId, errors.Err(err))
		}
	})

	r.Runner.HandleJobComplete(func(job runner.Job) {
		j := r.runnerJobs[job.Stage+job.Name]
		j.flush()
		j.finished = true

		r.runnerJobs[job.Stage+job.Name] = j

		if err := r.jobs.Finished(j.id, j.buf.String(), job.Status); err != nil {
			r.log.Error.Println("failed to handle job finish", jobId, errors.Err(err))
		}
	})

	if err := r.builds.Started(r.build.ID); err != nil {
		r.log.Error.Println(jobId, "failed to mark build as started", err)
		return runner.Failed, errors.Err(err)
	}

	r.Runner.Run(ctx, driver)

	if err := r.builds.Finished(r.build.ID, r.buf.String(), r.Runner.Status); err != nil {
		r.log.Error.Println(jobId, "failed to mark build as finished", err)
		return r.Runner.Status, errors.Err(err)
	}

	// Use different context for updating the jobs, as the build may have been
	// killed via the context passed to this method.
	if err := r.updateUnfinishedJobs(context.Background(), r.Runner.Status); err != nil {
		r.log.Error.Println(jobId, "failed to update build jobs", err)
		return r.Runner.Status, errors.Err(err)
	}
	return r.Runner.Status, nil
}
