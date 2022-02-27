package config

import (
	"io"
	"runtime"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/driver/qemu"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/log"
	"djinn-ci.com/mail"
	"djinn-ci.com/provider"

	"github.com/andrewpillar/config"

	"github.com/go-redis/redis"

	"github.com/mcmathja/curlyq"
)

type workerCfg struct {
	Pidfile string

	Parallelism int
	Driver      string
	Timeout     time.Duration

	Log map[string]string

	Crypto cryptoCfg

	SMTP smtpCfg

	Database databaseCfg
	Redis    redisCfg

	Store map[string]storeCfg

	Provider map[string]providerCfg
}

type Worker struct {
	pidfile string

	log *log.Logger

	driver      string
	queue       string
	parallelism int
	timeout     time.Duration

	consumer *curlyq.Consumer

	aesgcm *crypto.AESGCM

	db    database.Pool
	redis *redis.Client

	smtp      *mail.Client
	smtpadmin string

	artifacts fs.Store
	objects   fs.Store

	providers *provider.Registry
}

func (w *Worker) Pidfile() string               { return w.pidfile }
func (w *Worker) Log() *log.Logger              { return w.log }
func (w *Worker) Driver() string                { return w.driver }
func (w *Worker) Parallelism() int              { return w.parallelism }
func (w *Worker) Queue() string                 { return w.queue }
func (w *Worker) Consumer() *curlyq.Consumer    { return w.consumer }
func (w *Worker) Timeout() time.Duration        { return w.timeout }
func (w *Worker) DB() database.Pool             { return w.db }
func (w *Worker) Redis() *redis.Client          { return w.redis }
func (w *Worker) SMTP() (*mail.Client, string)  { return w.smtp, w.smtpadmin }
func (w *Worker) Artifacts() fs.Store           { return w.artifacts }
func (w *Worker) Objects() fs.Store             { return w.objects }
func (w *Worker) AESGCM() *crypto.AESGCM        { return w.aesgcm }
func (w *Worker) Providers() *provider.Registry { return w.providers }

func DecodeWorker(name string, r io.Reader) (*Worker, error) {
	var cfg workerCfg

	dec := config.NewDecoder(name, decodeOpts...)

	if err := dec.Decode(&cfg, r); err != nil {
		return nil, err
	}

	worker := &Worker{}

	var err error

	worker.pidfile, err = mkpidfile(cfg.Pidfile)

	if err != nil {
		return nil, err
	}

	worker.log, err = logger(cfg.Log)

	if err != nil {
		return nil, err
	}

	worker.driver = cfg.Driver

	worker.parallelism = cfg.Parallelism

	if worker.parallelism == 0 {
		worker.parallelism = int(runtime.NumCPU())
	}

	worker.timeout = cfg.Timeout

	worker.aesgcm, err = cfg.Crypto.aesgcm()

	if err != nil {
		return nil, err
	}

	worker.db, err = cfg.Database.connect(worker.log)

	if err != nil {
		return nil, err
	}

	worker.redis, err = cfg.Redis.connect(worker.log)

	if err != nil {
		return nil, err
	}

	worker.queue = defaultBuildQueue + "_" + worker.driver

	// With qemu drivers the builds are split up into different queues depending
	// on the arch that build wants to use. So modify the queue name with the
	// host arch.
	if worker.driver == "qemu" {
		arch, err := qemu.GetExpectedArch()

		if err != nil {
			return nil, err
		}
		worker.queue += "-" + arch
	}

	worker.consumer = curlyq.NewConsumer(&curlyq.ConsumerOpts{
		Queue:  worker.queue,
		Client: worker.redis,
		Logger: log.Queue{
			Logger: worker.log,
		},
		ProcessorConcurrency: worker.parallelism,
		JobMaxAttempts:       1,
	})

	worker.smtp, worker.smtpadmin, err = cfg.SMTP.connect(worker.log)

	if err != nil {
		return nil, err
	}

	for _, label := range []string{"artifacts", "objects"} {
		s, ok := cfg.Store[label]

		if !ok {
			return nil, errors.New(label + " store not configured")
		}

		var err error

		switch label {
		case "artifacts":
			worker.artifacts, err = s.store()
		case "objects":
			worker.objects, err = s.store()
		}

		if err != nil {
			return nil, err
		}
	}

	worker.providers = provider.NewRegistry()

	for name, p := range cfg.Provider {
		cli, err := p.client(name, "")

		if err != nil {
			return nil, err
		}
		worker.providers.Register(name, cli)
	}
	return worker, nil
}
