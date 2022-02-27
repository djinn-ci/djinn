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

	"djinn-ci.com/build"
	"djinn-ci.com/config"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/image"
	"djinn-ci.com/log"
	"djinn-ci.com/mail"
	"djinn-ci.com/namespace"
	"djinn-ci.com/provider"
	"djinn-ci.com/queue"
	"djinn-ci.com/runner"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/go-redis/redis"

	"github.com/mcmathja/curlyq"
)

type Worker struct {
	// DB is the client to the SQL database.
	DB database.Pool

	// Redis is the client to the Redis database from where submitted builds
	// will be processed from.
	Redis *redis.Client

	// SMTP is the client to the SMTP server we use for sending emails about
	// a build's progress.
	SMTP *mail.Client

	// Admin is the email address that should be used in emails sent from the
	// worker on build failures.
	Admin string

	// AESGCM is the block cipher to use for decrypting access tokens when
	// interacting with a provider's REST API.
	AESGCM *crypto.AESGCM

	// Log is the logger implementation used for logging information about the
	// running builds.
	Log *log.Logger

	// Consumer is the consumer used for retrieving builds from the queue.
	Consumer *curlyq.Consumer

	Queue queue.Queue // Queue for dispatching webhooks.

	Driver  string        // Driver is the name of the driver the worker is configured for.
	Timeout time.Duration // Timeout is the maximum duration a build can run for.

	DriverInit   driver.Init   // DriverInit is the initialization function for the worker's driver.
	DriverConfig driver.Config // DriverConfig is the global configuration for the worker's driver.

	// Providers is the registry containing the provider client implementations
	// used for updating a build's commit status, if submitted via a pull
	// request hook.
	Providers *provider.Registry

	Objects   fs.Store // The fs.Store from where we place build objects.
	Artifacts fs.Store // The fs.Store to where we collect build artifacts.
}

var passedStatuses = map[runner.Status]struct{}{
	runner.Queued:             {},
	runner.Running:            {},
	runner.Passed:             {},
	runner.PassedWithFailures: {},
}

func New(cfg *config.Worker, drivercfg driver.Config, fn driver.Init) *Worker {
	smtp, smtpadmin := cfg.SMTP()

	return &Worker{
		DB:           cfg.DB(),
		Redis:        cfg.Redis(),
		SMTP:         smtp,
		Admin:        smtpadmin,
		AESGCM:       cfg.AESGCM(),
		Log:          cfg.Log(),
		Consumer:     cfg.Consumer(),
		Timeout:      cfg.Timeout(),
		Driver:       cfg.Driver(),
		DriverInit:   fn,
		DriverConfig: drivercfg,
		Providers:    cfg.Providers(),
		Objects:      cfg.Objects(),
		Artifacts:    cfg.Artifacts(),
	}
}

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

	w.Log.Debug.Println("build id:", payload.BuildID)

	builds := build.Store{Pool: w.DB}

	b, _, err := builds.Get(query.Where("id", "=", query.Arg(payload.BuildID)))

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	if b.FinishedAt.Valid {
		return nil
	}

	users := user.Store{Pool: w.DB}

	b.User, _, err = users.Get(query.Where("id", "=", query.Arg(b.UserID)))

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	triggers := build.TriggerStore{Pool: w.DB}

	b.Trigger, _, err = triggers.Get()

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	namespaces := namespace.Store{Pool: w.DB}

	b.Namespace, _, err = namespaces.Get(query.Where("id", "=", query.Arg(b.NamespaceID)))

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
	}

	if b.Namespace != nil {
		b.Namespace.User, _, err = users.Get(query.Where("id", "=", query.Arg(b.Namespace.UserID)))

		if err != nil {
			w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		}
	}

	providers := provider.Store{
		Pool:    w.DB,
		AESGCM:  w.AESGCM,
		Clients: w.Providers,
	}

	p, fromProvider, err := providers.Get(query.Where("id", "=", query.Arg(b.Trigger.ProviderID)))

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	var repo *provider.Repo

	if fromProvider {
		repos := provider.RepoStore{Pool: w.DB}

		r, _, err := repos.Get(
			query.Where("id", "=", query.Arg(b.Trigger.RepoID)),
			query.Where("provider_id", "=", query.Arg(p.ID)),
		)

		if err != nil {
			w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
			return errors.Err(err)
		}

		repo = r

		if b.Trigger.Type == build.Pull {
			url := payload.Host + b.Endpoint()

			if err := p.SetCommitStatus(repo, runner.Running, url, b.Trigger.Data["sha"]); err != nil {
				w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
				return errors.Err(err)
			}
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

	drivers := build.DriverStore{Pool: w.DB}

	d, _, err := drivers.Get(query.Where("build_id", "=", query.Arg(b.ID)))

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	b.Status = runner.Running

	w.Queue.Produce(ctx, &build.Event{
		Build: b,
	})

	status, err := run.Run(ctx, job.ID, d)

	if err != nil {
		w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
		return errors.Err(err)
	}

	b.Status = status

	w.Queue.Produce(ctx, &build.Event{Build: b})

	if fromProvider {
		if b.Trigger.Type == build.Pull {
			url := payload.Host + b.Endpoint()

			if err := p.SetCommitStatus(repo, status, url, b.Trigger.Data["sha"]); err != nil {
				w.Log.Error.Println(job.ID, "build_id =", payload.BuildID, errors.Err(err))
				return errors.Err(err)
			}
		}
	}

	if w.SMTP == nil {
		return nil
	}

	if _, ok := passedStatuses[status]; ok {
		return nil
	}

	to := []string{b.User.Email}

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
	buf.WriteString(b.Trigger.String())
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

	if err := w.Consumer.ConsumeCtx(ctx, w.handle); err != nil {
		return errors.Err(err)
	}
	return nil
}

// Runner configures a new runner for running the given build.
func (w *Worker) Runner(b *build.Build) *Runner {
	return &Runner{
		db:     w.DB,
		aesgcm: w.AESGCM,
		log:    w.Log,
		buf:    &bytes.Buffer{},
		build:  b,
		builds: &build.Store{Pool: w.DB},
		images: &image.Store{Pool: w.DB},
		vars:   build.VariableStore{Pool: w.DB},
		keys:   build.KeyStore{Pool: w.DB},
		stages: build.StageStore{Pool: w.DB},
		jobs:   build.JobStore{Pool: w.DB},
		objects: &build.ObjectStore{
			Pool:  w.DB,
			Store: w.Objects,
		},
		artifacts: &build.ArtifactStore{
			Pool:  w.DB,
			Store: w.Artifacts,
		},
		driver:     w.Driver,
		driverinit: w.DriverInit,
		drivercfg:  w.DriverConfig,
	}
}
