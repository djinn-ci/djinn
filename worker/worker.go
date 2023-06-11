package worker

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"net/url"
	"runtime/debug"
	"time"

	"djinn-ci.com/build"
	"djinn-ci.com/config"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/log"
	"djinn-ci.com/mail"
	"djinn-ci.com/namespace"
	"djinn-ci.com/provider"
	"djinn-ci.com/queue"
	"djinn-ci.com/runner"
	"djinn-ci.com/user"

	"github.com/andrewpillar/fs"
	"github.com/andrewpillar/query"

	"github.com/go-redis/redis"

	"github.com/mcmathja/curlyq"
)

type Worker struct {
	Log *log.Logger

	DB    *database.Pool
	Redis *redis.Client

	SMTP *mail.Client

	AESGCM *crypto.AESGCM

	Consumer *curlyq.Consumer
	Queue    queue.Queue

	Driver  string
	Timeout time.Duration

	DriverInit   driver.Init
	DriverConfig driver.Config

	Providers *provider.Registry

	Objects       fs.FS
	Artifacts     fs.FS
	ArtifactLimit int64
}

func New(cfg *config.Worker, driverCfg driver.Config, driverInit driver.Init) *Worker {
	smtp, _ := cfg.SMTP()

	aesgcm := cfg.AESGCM()
	log := cfg.Log()

	webhooks := &namespace.WebhookStore{
		Store:  namespace.NewWebhookStore(cfg.DB()),
		AESGCM: aesgcm,
	}

	memq := queue.NewMemory(cfg.Parallelism(), func(j queue.Job, err error) {
		log.Error.Println("queue job failed:", j.Name(), err)
	})

	memq.InitFunc("event:build.started", build.InitEvent(webhooks))
	memq.InitFunc("event:build.finished", build.InitEvent(webhooks))

	return &Worker{
		Log:           log,
		DB:            cfg.DB(),
		Redis:         cfg.Redis(),
		SMTP:          smtp,
		AESGCM:        aesgcm,
		Consumer:      cfg.Consumer(),
		Driver:        cfg.Driver(),
		Queue:         memq,
		Timeout:       cfg.Timeout(),
		DriverInit:    driverInit,
		DriverConfig:  driverCfg,
		Providers:     cfg.Providers(),
		Objects:       cfg.Objects(),
		Artifacts:     cfg.Artifacts(),
		ArtifactLimit: cfg.ArtifactLimit(),
	}
}

func (w *Worker) SetCommitStatus(ctx context.Context, p *provider.Provider, payload build.Payload, b *build.Build) error {
	r, _, err := provider.NewRepoStore(w.DB).Get(
		ctx,
		query.Where("id", "=", query.Arg(b.Trigger.RepoID)),
		query.Where("provider_id", "=", query.Arg(p.ID)),
	)

	if err != nil {
		return errors.Err(err)
	}

	if b.Trigger.Type == build.Pull {
		cli := p.Client()
		url := payload.Host + b.Endpoint()

		if err := cli.SetCommitStatus(r, b.Status, url, b.Trigger.Data["sha"]); err != nil {
			return errors.Err(err)
		}
	}
	return nil
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

	data := bytes.NewBuffer(job.Data)

	var payload build.Payload

	if err := gob.NewDecoder(data).Decode(&payload); err != nil {
		return errors.Err(err)
	}

	w.Log.Debug.Println("received build", payload.BuildID)

	builds := build.Store{
		Store: build.NewStore(w.DB),
	}

	b, _, err := builds.SelectOne(
		ctx,
		[]string{"id", "user_id", "number", "output", "status", "secret", "namespace_id", "started_at", "finished_at"},
		query.Where("id", "=", query.Arg(payload.BuildID)),
	)

	if err != nil {
		return errors.Err(err)
	}

	if b.StartedAt.Valid {
		w.Log.Debug.Println("orphaning build", b.ID)

		if err := builds.Orphan(ctx, b); err != nil {
			return errors.Err(err)
		}
	}

	if b.FinishedAt.Valid {
		return nil
	}

	if err := build.LoadRelations(ctx, w.DB, b); err != nil {
		return errors.Err(err)
	}

	providers := &provider.Store{
		Store:   provider.NewStore(w.DB),
		AESGCM:  w.AESGCM,
		Clients: w.Providers,
	}

	p, fromProvider, err := providers.Get(ctx, query.Where("id", "=", query.Arg(b.Trigger.ProviderID)))

	if err != nil {
		return errors.Err(err)
	}

	if fromProvider {
		if err := w.SetCommitStatus(ctx, p, payload, b); err != nil {
			return errors.Err(err)
		}
	}

	b.Status = runner.Running

	w.Queue.Produce(ctx, &build.Event{Build: b})

	r, err := NewRunner(ctx, w, b)

	if err != nil {
		return errors.Err(err)
	}

	if err := r.Run(ctx); err != nil {
		return errors.Err(err)
	}

	w.Queue.Produce(ctx, &build.Event{Build: b})

	if fromProvider {
		if err := w.SetCommitStatus(ctx, p, payload, b); err != nil {
			return errors.Err(err)
		}
	}

	if w.SMTP != nil {
		passed := map[runner.Status]struct{}{
			runner.Queued:             {},
			runner.Running:            {},
			runner.Passed:             {},
			runner.PassedWithFailures: {},
		}

		if _, ok := passed[b.Status]; !ok {
			if err := w.SendEmail(ctx, r, payload, b); err != nil {
				return errors.Err(err)
			}
		}
	}
	return nil
}

const emailTmpl = `Build: %s%s

-----
%s
-----

%s
`

func (w *Worker) SendEmail(ctx context.Context, r *Runner, payload build.Payload, b *build.Build) error {
	to := make([]string, 0)
	to = append(to, b.User.Email)

	if b.NamespaceID.Valid {
		uu, err := user.NewStore(w.DB).Select(
			ctx,
			[]string{"email"},
			query.Where("id", "IN",
				namespace.SelectCollaborator(
					query.Columns("user_id"),
					query.Where("namespace_id", "=", query.Arg(b.NamespaceID)),
				),
			),
		)

		if err != nil {
			return errors.Err(err)
		}

		for _, u := range uu {
			if u.Email != "" {
				to = append(to, u.Email)
			}
		}
	}

	var buf bytes.Buffer

	fmt.Fprintf(&buf, emailTmpl, payload.Host, b.Endpoint(), b.Trigger.String(), r.Tail())

	host := payload.Host

	url, err := url.Parse(host)

	if err == nil {
		host, _, _ = net.SplitHostPort(url.Host)
	}

	m := mail.Mail{
		From:    "djinn-worker@" + host,
		To:      to,
		Subject: fmt.Sprintf("Djinn CI - Build #%d %s", b.Number, b.Status.String()),
		Body:    buf.String(),
	}

	w.Log.Debug.Println("sending mail from", m.From)
	w.Log.Debug.Println("sending mail to", m.To)

	if err := m.Send(w.SMTP); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (w *Worker) Run(ctx context.Context) error {
	gob.Register(build.Payload{})

	handle := func(ctx context.Context, job curlyq.Job) error {
		if err := w.handle(ctx, job); err != nil {
			w.Log.Error.Println(errors.Err(err))
			return err
		}
		return nil
	}

	if err := w.Consumer.ConsumeCtx(ctx, handle); err != nil {
		return errors.Err(err)
	}
	return nil
}
