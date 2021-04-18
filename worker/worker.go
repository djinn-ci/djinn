// Package worker contains the Worker implementation that is used for handling
// builds that are submitted to the server.
package worker

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/driver"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/fs"
	"github.com/andrewpillar/djinn/log"
	"github.com/andrewpillar/djinn/mail"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/provider"
	"github.com/andrewpillar/djinn/runner"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/mcmathja/curlyq"
)

type workerLogger struct {
	log *log.Logger
}

type Worker struct {
	// DB is the client to the SQL database.
	DB *sqlx.DB

	// Redis is the client to the Redis database from where submitted builds
	// will be processed from.
	Redis *redis.Client

	// SMTP is the client to the SMTP server we use for sending emails about
	// a build's progress.
	SMTP *mail.Client

	// Admin is the email address that should be used in emails sent from the
	// worker on build failures.
	Admin string

	// Block is the block cipher to use for decrypting access tokens when
	// interacting with a provider's REST API.
	Block *crypto.Block

	// Log is the logger implementation used for logging information about the
	// running builds.
	Log *log.Logger

	Driver      string        // Driver is the name of the driver the worker is configured for.
	Queue       string        // Queue is the name of the queue the worker should work from.
	Parallelism int           // Parallelism is how many builds should be processed at once.
	Timeout     time.Duration // Timeout is the maximum duration a build can run for.

	Init   driver.Init   // Init is the initialization function for the worker's driver.
	Config driver.Config // Config is the global configuration for the worker's driver.

	// Providers is the registry containing the provider client implementations
	// used for updating a build's commit status, if submitted via a pull
	// request hook.
	Providers *provider.Registry

	Objects   fs.Store // The fs.Store from where we place build objects.
	Artifacts fs.Store // The fs.Store to where we collect build artifacts.
}

var (
	_ curlyq.Logger = (*workerLogger)(nil)

	passedStatuses = map[runner.Status]struct{}{
		runner.Queued:             {},
		runner.Running:            {},
		runner.Passed:             {},
		runner.PassedWithFailures: {},
	}
)

func (l workerLogger) Debug(v ...interface{}) { l.log.Debug.Println(v...) }
func (l workerLogger) Info(v ...interface{})  { l.log.Info.Println(v...) }
func (l workerLogger) Warn(v ...interface{})  { l.log.Warn.Println(v...) }
func (l workerLogger) Error(v ...interface{}) { l.log.Error.Println(v...) }

func (w *Worker) handle(ctx context.Context, job curlyq.Job) error {
	defer func() {
		if v := recover(); v != nil {
			if err, ok := v.(error); ok {
				w.Log.Error.Println(err)
			}
			w.Log.Error.Println(string(debug.Stack()))
		}
	}()

	if err := ctx.Err(); err != nil {
		return errors.Err(err)
	}

	w.Log.Debug.Println("handling job", job.ID, "attempt:", job.Attempt)

	data := bytes.NewBuffer(job.Data)

	var payload build.Payload

	if err := gob.NewDecoder(data).Decode(&payload); err != nil {
		w.Log.Error.Println(job.ID, "failed to decode job payload", errors.Err(err))
		return errors.Err(err)
	}

	b, err := build.NewStore(w.DB).Get(query.Where("id", "=", query.Arg(payload.BuildID)))

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	if b.FinishedAt.Valid {
		return nil
	}

	b.User, err = user.NewStore(w.DB).Get(query.Where("id", "=", query.Arg(b.UserID)))

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	t, err := build.NewTriggerStore(w.DB, b).Get()

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	p, err := provider.NewStore(w.DB).Get(query.Where("id", "=", query.Arg(t.ProviderID)))

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	r, err := provider.NewRepoStore(w.DB, p).Get(query.Where("id", "=", query.Arg(t.RepoID)))

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	if t.Type == build.Pull {
		err := p.SetCommitStatus(
			w.Block,
			w.Providers,
			r,
			runner.Running,
			payload.Host+b.Endpoint(),
			t.Data["sha"],
		)

		if err != nil {
			w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
			return errors.Err(err)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, w.Timeout)
	defer cancel()

	sub := w.Redis.Subscribe(fmt.Sprintf("kill-%v", b.ID))
	defer sub.Close()

	go func() {
		if msg := <-sub.Channel(); msg != nil {
			if msg.Payload == b.Secret.String {
				cancel()
			}
		}
	}()

	run := w.Runner(b)

	if err := run.Init(); err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	d, err := build.NewDriverStore(w.DB, b).Get()

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	status, err := run.Run(ctx, job.ID, d)

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	if t.Type == build.Pull {
		err := p.SetCommitStatus(
			w.Block,
			w.Providers,
			r,
			status,
			payload.Host+b.Endpoint(),
			t.Data["sha"],
		)

		if err != nil {
			w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
			return errors.Err(err)
		}
	}

	if w.SMTP == nil {
		return nil
	}

	if _, ok := passedStatuses[status]; ok {
		return nil
	}

	to := make([]string, 0)

	users := user.NewStore(w.DB)

	u, err := users.Get(query.Where("id", "=", query.Arg(b.UserID)))

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	b.User = u

	to = append(to, u.Email)

	if b.NamespaceID.Valid {
		uu, err := users.All(
			query.Where("id", "IN",
				namespace.CollaboratorSelect("user_id",
					query.Where("namespace_id", "=", query.Arg(b.NamespaceID)),
				),
			),
		)

		if err != nil {
			w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
			return errors.Err(err)
		}

		for _, u := range uu {
			to = append(to, u.Email)
		}
	}

	var (
		buf    bytes.Buffer
		output string
	)

	adj := strings.Replace(status.String(), "_", " ", -1)
	subj := fmt.Sprintf("Djinn CI - Build #%d %s", b.Number, adj)

	if status == runner.Failed {
		output = run.Tail()
	} else {
		buf.WriteString(subj + "\n\n")
	}

	fmt.Fprintf(&buf, "Build: %s%s\n\n", payload.Host, b.Endpoint())
	buf.WriteString("-----\n")
	buf.WriteString(t.String())
	buf.WriteString("-----\n")

	if output != "" {
		buf.WriteString("\n" + output + "\n")
	}

	m := mail.Mail{
		From:    w.Admin,
		To:      to,
		Subject: subj,
		Body:    buf.String(),
	}

	if err := m.Send(w.SMTP); err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}
	return nil
}

// Run begins the worker for handling builds.
func (w *Worker) Run(ctx context.Context) error {
	gob.Register(build.Payload{})

	consumer := curlyq.NewConsumer(&curlyq.ConsumerOpts{
		Queue:                w.Queue,
		Client:               w.Redis,
		Logger:               workerLogger{log: w.Log},
		ProcessorConcurrency: w.Parallelism,
		JobMaxAttempts:       1,
	})
	return errors.Err(consumer.ConsumeCtx(ctx, w.handle))
}

// Runner configures a new runner for running the given build.
func (w *Worker) Runner(b *build.Build) *Runner {
	return &Runner{
		db:        w.DB,
		block:     w.Block,
		log:       w.Log,
		build:     b,
		objects:   w.Objects,
		artifacts: w.Artifacts,
		driver:    w.Driver,
		init:      w.Init,
		config:    w.Config,
		buf:       &bytes.Buffer{},
		bufs:      make(map[int64]*bytes.Buffer),
		jobs:      make(map[string]*build.Job),
	}
}
