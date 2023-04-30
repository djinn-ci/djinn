package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/build"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/driver/qemu"
	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/key"
	"djinn-ci.com/log"
	"djinn-ci.com/runner"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/query"

	"github.com/go-redis/redis"

	"golang.org/x/text/transform"
)

type maskedBuffer struct {
	*transform.Writer

	buf *bytes.Buffer
}

func newBuffer(t transform.Transformer) *maskedBuffer {
	var buf bytes.Buffer

	return &maskedBuffer{
		Writer: transform.NewWriter(&buf, t),
		buf:    &buf,
	}
}

func (b *maskedBuffer) String() string { return b.buf.String() }

type job struct {
	job *build.Job
	buf *maskedBuffer
}

func newJob(j *build.Job, t transform.Transformer) *job {
	return &job{
		job: j,
		buf: newBuffer(t),
	}
}

func (j *job) fullname() string {
	return j.job.Stage.Name + "/" + j.job.Name
}

func (j *job) runnerJob(r *runner.Runner) *runner.Job {
	artifacts := make(runner.Passthrough)

	for _, a := range j.job.Artifacts {
		artifacts[a.Source] = a.Name
	}

	return &runner.Job{
		Writer:    io.MultiWriter(r.Writer, j.buf),
		Name:      j.job.Name,
		Commands:  strings.Split(j.job.Commands, "\n"),
		Artifacts: artifacts,
	}
}

type jobs struct {
	build.JobStore

	tab map[string]*job
}

func (s *jobs) put(j *job) {
	s.tab[j.fullname()] = j
}

func (s *jobs) get(j *runner.Job) *job {
	return s.tab[j.FullName()]
}

type Runner struct {
	*runner.Runner

	timeout time.Duration
	redis   *redis.Client

	log *log.Logger

	buf *maskedBuffer

	driver     string
	driverInit driver.Init
	driverCfg  driver.Config

	builds *build.Store
	images *database.Store[*image.Image]

	jobs  *jobs
	build *build.Build
}

func NewRunner(ctx context.Context, w *Worker, b *build.Build) (*Runner, error) {
	vv, err := build.NewVariableStore(w.DB).All(ctx, query.Where("build_id", "=", query.Arg(b.ID)))

	if err != nil {
		return nil, errors.Err(err)
	}

	chain := make([]transform.Transformer, 0, len(vv))
	env := make([]string, 0, len(vv))

	for _, v := range vv {
		if err := variable.Unmask(w.AESGCM, v.Variable); err != nil {
			return nil, errors.Err(err)
		}

		env = append(env, v.Variable.String())

		if v.Masked {
			chain = append(chain, variable.Masker(v.Value))
		}
	}

	masker := transform.Chain(chain...)

	kk, err := build.NewKeyStore(w.DB).All(ctx, query.Where("build_id", "=", query.Arg(b.ID)))

	if err != nil {
		return nil, errors.Err(err)
	}

	pt := make(runner.Passthrough)

	if len(kk) > 0 {
		for _, k := range kk {
			pt[k.Name] = k.Location
		}
		pt["/root/.ssh/config"] = "/root/.ssh/config"
	}

	oo, err := build.NewObjectStore(w.DB).All(ctx, query.Where("build_id", "=", query.Arg(b.ID)))

	if err != nil {
		return nil, errors.Err(err)
	}

	for _, o := range oo {
		pt[o.Source] = o.Name
	}

	objects := build.ObjectStore{
		Store: build.NewObjectStore(w.DB),
		FS:    w.Objects,
	}

	artifacts := build.ArtifactStore{
		Store: build.NewArtifactStore(w.DB),
		FS:    w.Artifacts,
	}

	jj, err := build.NewJobStore(w.DB).All(ctx, query.Where("build_id", "=", query.Arg(b.ID)), query.OrderAsc("created_at"))

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := build.LoadJobRelations(ctx, w.DB, jj...); err != nil {
		return nil, errors.Err(err)
	}

	r := &Runner{
		timeout:    w.Timeout,
		redis:      w.Redis,
		log:        w.Log,
		buf:        newBuffer(masker),
		driver:     w.Driver,
		driverInit: w.DriverInit,
		driverCfg:  w.DriverConfig,
		builds: &build.Store{
			Store: build.NewStore(w.DB),
		},
		images: image.NewStore(w.DB),
		jobs: &jobs{
			JobStore: build.JobStore{
				Store: build.NewJobStore(w.DB),
			},
			tab: make(map[string]*job, len(jj)),
		},
		build: b,
	}

	r.Runner = &runner.Runner{
		Writer:      r.buf,
		Env:         env,
		Passthrough: pt,
		Objects:     objects.Filestore(b, keyChain(w.AESGCM, kk)),
		Artifacts:   artifacts.Filestore(b),
	}

	ss, err := build.NewStageStore(w.DB).Select(
		ctx,
		[]string{"id", "name", "can_fail"},
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.OrderAsc("created_at"),
	)

	if err != nil {
		return nil, errors.Err(err)
	}

	stagetab := make(map[int64]*runner.Stage, len(ss))

	for _, s := range ss {
		st := &runner.Stage{
			Name:    s.Name,
			CanFail: s.CanFail,
		}

		stagetab[s.ID] = st
		r.Runner.Add(st)
	}

	for _, j := range jj {
		jb := newJob(j, masker)
		rj := jb.runnerJob(r.Runner)

		stagetab[j.StageID].Add(rj)
		r.jobs.put(jb)
	}
	return r, nil
}

func keyChain(aesgcm *crypto.AESGCM, buildKeys []*build.Key) *key.Chain {
	kk := make([]*key.Key, 0, len(buildKeys))

	for _, k := range buildKeys {
		kk = append(kk, &key.Key{
			Name:   k.Name,
			Key:    k.Key,
			Config: k.Config,
		})
	}
	return key.NewChain(aesgcm, kk)
}

func (r *Runner) qemuRealpath(ctx context.Context, dir string) qemu.RealpathFunc {
	return func(arch, name string) (string, error) {
		col := "user_id"
		arg := r.build.UserID

		if r.build.NamespaceID.Valid {
			col = "namespace_id"
			arg = r.build.NamespaceID.Elem
		}

		i, ok, err := r.images.Get(
			ctx, query.Where(col, "=", query.Arg(arg)), query.Where("name", "=", query.Arg(name)),
		)

		if err != nil {
			return "", errors.Err(err)
		}

		if !ok {
			return filepath.Join(dir, "_base", "qemu", arch, name), nil
		}
		return filepath.Join(dir, strconv.Itoa(int(i.UserID)), "qemu", i.Hash), nil
	}
}

func sanitize(s string) string {
	buf := make([]rune, 0, len(s))

	for _, r := range s {
		if r == 0 {
			buf = append(buf, 'N', 'U', 'L')
		}
	}
	return string(buf)
}

func (r *Runner) Run(ctx context.Context) error {
	cfg := r.driverCfg.Merge(r.build.Driver.Config)
	d := r.driverInit(r.buf.buf, cfg)

	if q, ok := d.(*qemu.Driver); ok {
		qemuCfg := cfg.(*qemu.Config)

		q.Image = filepath.Clean(q.Image)
		q.Realpath = r.qemuRealpath(ctx, qemuCfg.Disks)
	}

	r.HandleJobStart(func(rj *runner.Job) {
		j := r.jobs.get(rj)
		j.job.Status = rj.Status()

		if err := r.jobs.Started(ctx, j.job); err != nil {
			r.log.Error.Println(errors.Err(err))
		}
	})

	r.HandleJobComplete(func(rj *runner.Job) {
		j := r.jobs.get(rj)
		j.buf.Close()

		j.job.Output = database.Null[string]{
			Elem:  sanitize(j.buf.String()),
			Valid: true,
		}
		j.job.Status = rj.Status()

		if err := r.jobs.Finished(ctx, j.job); err != nil {
			r.log.Error.Println(errors.Err(err))
		}
	})

	if err := r.builds.Started(ctx, r.build); err != nil {
		return errors.Err(err)
	}

	r.log.Debug.Println("running build", r.build.ID)

	timeoutCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	sub := r.redis.Subscribe(fmt.Sprintf("kill-%v", r.build.ID))
	defer sub.Close()

	go func() {
		if msg := <-sub.Channel(); msg != nil {
			r.log.Debug.Println("kill signal received for build", r.build.ID)

			if msg.Payload == r.build.Secret.Elem {
				r.log.Debug.Println("killing build", r.build.ID)
				cancel()
			}
		}
	}()

	if err := r.Runner.Run(timeoutCtx, d); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			return errors.Err(err)
		}
	}

	r.log.Debug.Println("build finished", r.build.ID)

	r.buf.Close()

	r.build.Output = database.Null[string]{
		Elem:  sanitize(r.buf.String()),
		Valid: true,
	}
	r.build.Status = r.Status()

	r.log.Debug.Println("setting build", r.build.ID, "status to", r.build.Status)

	if err := r.builds.Finished(ctx, r.build); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (r *Runner) Tail() string {
	parts := strings.Split(r.buf.String(), "\n")

	if len(parts) >= 15 {
		parts = parts[len(parts)-15:]
	}
	return strings.Join(parts, "\n")
}
