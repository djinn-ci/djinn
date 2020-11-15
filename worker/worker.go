// Package worker contains the Worker implementation that is used for handling
// builds that are submitted to the server.
package worker

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/driver"
	"github.com/andrewpillar/djinn/errors"
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
	SMTP *smtp.Client

	// Admin is the email address that should be used in emails sent from the
	// worker on build failures.
	Admin string

	// Block is the block cipher to use for decrypting access tokens when
	// interacting with a provider's REST API.
	Block *crypto.Block

	// Log is the logger implementation used for logging information about the
	// running builds.
	Log *log.Logger

	Queue       string        // Queue is the name of the queue the worker should work from.
	Parallelism int           // Parallelism is how many builds should be processed at once.
	Timeout     time.Duration // Timeout is the maximum duration a build can run for.

	// Config is the global driver configuration values. This will store
	// information about the various drivers, such as where the QEMU images are
	// stored, or the Docker API version to use.
	Config map[string]map[string]interface{}

	// Drivers is the registry containg the driver implementations the worker
	// can use for running builds.
	Drivers *driver.Registry

	// Providers is the registry containing the provider client implementations
	// used for updating a build's commit status, if submitted via a pull
	// request hook.
	Providers *provider.Registry

	Placer    runner.Placer    // Placer is the implementation used for placing build objects.
	Collector runner.Collector // Collector is the implementation used for collecting artifacts.
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
	if err := ctx.Err(); err != nil {
		return errors.Err(err)
	}

	w.Log.Debug.Println("handling job", job.ID, "attempt:", job.Attempt)

	data := bytes.NewBuffer(job.Data)

	var payload build.Payload

	if err := gob.NewDecoder(data).Decode(&payload); err != nil {
		return errors.Err(err)
	}

	b := &build.Build{
		ID:   payload.BuildID,
		User: &payload.User,
	}

	t, err := build.NewTriggerStore(w.DB, b).Get()

	if err != nil {
		return errors.Err(err)
	}

	p, err := provider.NewStore(w.DB).Get(query.Where("id", "=", query.Arg(t.ProviderID)))

	if err != nil {
		return errors.Err(err)
	}

	r, err := provider.NewRepoStore(w.DB, p).Get(query.Where("id", "=", query.Arg(t.RepoID)))

	if err != nil {
		return errors.Err(err)
	}

	if t.Type == build.Pull {
		err := p.SetCommitStatus(
			w.Block,
			w.Providers,
			r,
			runner.Running,
			host+b.Endpoint(),
			t.Data["sha"],
		)

		if err != nil {
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
		return errors.Err(err)
	}

	d, err := build.NewDriverStore(w.DB).Get()

	if err != nil {
		return errors.Err(err)
	}

	status, err := run.Run(ctx, d)

	if err != nil {
		return errors.Err(err)
	}

	if t.Type == build.Pull {
		err := p.SetCommitStatus(
			w.Block,
			w.Providers,
			r,
			status,
			host+b.Endpoint(),
			t.Data["sha"],
		)

		if err != nil {
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
	subj := fmt.Sprintf("Djinn - Build #%d %s", b.ID, adj)

	if status == runner.Failed {
		j, err := build.NewJobStore(w.DB, b).Get(
			query.Where("status", "=", query.Arg(runner.Failed)),
			query.OrderDesc("finished_at"),
		)

		if err != nil {
			return errors.Err(err)
		}
		fmt.Fprintf(&buf, "Job %s failed in build #%d failed\n\n", j.Name, b.ID)

		parts := strings.Split(j.Output.String, "\n")

		if len(parts) >= 15 {
			parts = parts[len(parts)-15:]
		}
		output = strings.Join(parts, "\n")
	} else {
		buf.WriteString(subj + "\n\n")
	}

	fmt.Fprintf(&buf, "Build: %s/%s\n\n", host, b.Endpoint())
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
	return errors.Err(m.Send(w.SMTP))
}

// Run begins the worker for handling builds.
func (w *Worker) Run(ctx context.Context) error {
	gob.Register(build.Build{})

	consumer := curlyq.NewConsumer(&curlyq.ConsumerOpts{
		Queue:                w.Queue,
		Client:               w.Redis,
		Logger:               workerLogger{log: w.Log},
		ProcessorConcurrency: w.Parallelism,
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
		placer:    w.Placer,
		collector: w.Collector,
		drivers:   w.Drivers,
		config:    w.Config,
		buf:       &bytes.Buffer{},
		bufs:      make(map[int64]*bytes.Buffer),
		jobs:      make(map[string]*build.Job),
	}
}
