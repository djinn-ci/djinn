package config

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/driver/qemu"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/log"
	"djinn-ci.com/mail"
	"djinn-ci.com/provider"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/mcmathja/curlyq"
)

type workerLogger struct {
	log *log.Logger
}

type workerCfg struct {
	Pidfile     string
	Parallelism int
	Driver      string
	Timeout     string

	Log logCfg

	Crypto Crypto

	SMTP smtpCfg

	Database databaseCfg
	Redis    redisCfg

	Stores    map[string]storeCfg
	Providers map[string]providerCfg
}

type Worker struct {
	pidfile *os.File

	log *log.Logger

	driver      string
	queue       string
	parallelism int
	timeout     time.Duration

	consumer *curlyq.Consumer

	block *crypto.Block

	db         *sqlx.DB
	redis      *redis.Client
	smtp       *mail.Client
	postmaster string

	artifacts fs.Store
	objects   fs.Store

	providers *provider.Registry
}

var _ curlyq.Logger = (*workerLogger)(nil)

func DecodeWorker(name string, r io.Reader) (*Worker, error) {
	errh := func(name string, line, col int, msg string) {
		fmt.Fprintf(os.Stderr, "%s,%d:%d - %s\n", name, line, col, msg)
	}

	p := newParser(name, r, errh)

	nodes := p.parse()

	if err := p.err(); err != nil {
		return nil, err
	}

	var cfg0 workerCfg

	for _, n := range nodes {
		if err := cfg0.put(n); err != nil {
			return nil, err
		}
	}

	var err error

	cfg := &Worker{}
	cfg.pidfile, err = mkpidfile(cfg0.Pidfile)

	if err != nil {
		return nil, err
	}

	cfg.log = log.New(os.Stdout)
	cfg.log.SetLevel(cfg0.Log.Level)

	if cfg0.Log.File != "/dev/stdout" {
		f, err := os.OpenFile(cfg0.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0640)

		if err != nil {
			return nil, err
		}
		cfg.log.SetWriter(f)
	}

	cfg.log.Info.Println("logging initialized, writing to", cfg0.Log.File)

	cfg.driver = cfg0.Driver

	if cfg.driver == "qemu" {
		cfg.driver += "-" + qemu.GetExpectedArch()
	}

	cfg.parallelism = cfg0.Parallelism

	if cfg.parallelism == 0 {
		cfg.parallelism = int(runtime.NumCPU())
	}

	cfg.timeout, err = time.ParseDuration(cfg0.Timeout)

	if err != nil {
		return nil, err
	}

	cfg.block, err = crypto.NewBlock(cfg0.Crypto.Block, cfg0.Crypto.Salt)

	if err != nil {
		return nil, err
	}

	cfg.db, err = connectdb(cfg.log, cfg0.Database)

	if err != nil {
		return nil, err
	}

	cfg.redis, err = connectredis(cfg.log, cfg0.Redis)

	if err != nil {
		return nil, err
	}

	cfg.queue = defaultBuildQueue + "_" + cfg.driver

	cfg.consumer = curlyq.NewConsumer(&curlyq.ConsumerOpts{
		Queue:                cfg.queue,
		Client:               cfg.redis,
		Logger:               workerLogger{log: cfg.log},
		ProcessorConcurrency: cfg.parallelism,
		JobMaxAttempts:       1,
	})

	cfg.smtp, err = connectsmtp(cfg.log, cfg0.SMTP)

	if err != nil {
		return nil, err
	}

	cfg.postmaster = cfg0.SMTP.Admin

	store, ok := cfg0.Stores["artifacts"]

	if !ok {
		return nil, errors.New("artifact store not configured")
	}

	cfg.artifacts = blockstores[store.Type](store.Path, store.Limit)

	store, ok = cfg0.Stores["objects"]

	if !ok {
		return nil, errors.New("object store not configured")
	}

	cfg.objects = blockstores[store.Type](store.Path, store.Limit)

	if err := cfg.artifacts.Init(); err != nil {
		return nil, err
	}
	if err := cfg.objects.Init(); err != nil {
		return nil, err
	}

	cfg.providers = provider.NewRegistry()

	for name, prv := range cfg0.Providers {
		fn, ok := providerFactories[name]

		if !ok {
			return nil, errors.New("unknown provider: " + name)
		}
		cfg.providers.Register(name, fn("", prv.Endpoint, prv.Secret, prv.ClientID, prv.ClientSecret))
	}
	return cfg, nil
}

func (l workerLogger) Debug(v ...interface{}) { l.log.Debug.Println(v...) }
func (l workerLogger) Info(v ...interface{})  { l.log.Info.Println(v...) }
func (l workerLogger) Warn(v ...interface{})  { l.log.Warn.Println(v...) }
func (l workerLogger) Error(v ...interface{}) { l.log.Error.Println(v...) }

func (w *workerCfg) put(n *node) error {
	switch n.name {
	case "pidfile":
		if n.lit != stringLit {
			return n.err("pidfile must be a string")
		}
		w.Pidfile = n.value
	case "parallelism":
		if n.lit != numberLit {
			return n.err("parallelism must be an integer")
		}

		i, err := strconv.ParseInt(n.value, 10, 64)

		if err != nil {
			return n.err("parallelism is not a valid integer")
		}
		w.Parallelism = int(i)
	case "driver":
		if n.lit != stringLit {
			return n.err("driver must be a string")
		}
		w.Driver = n.value
	case "timeout":
		if n.lit != stringLit {
			return n.err("timeout must be a duration string")
		}

		if n.value == "" {
			n.value = "30m"
		}
		w.Timeout = n.value
	case "log":
		return w.Log.put(n)
	case "crypto":
		// These values are not needed by the worker itself, but necessary to
		// pass validation, so spoof for now.
		w.Crypto.Hash = []byte("00000000000000000000000000000000")
		w.Crypto.Auth = []byte("00000000000000000000000000000000")

		if err := w.Crypto.put(n); err != nil {
			return err
		}

		w.Crypto.Hash = nil
		w.Crypto.Auth = nil
		return nil
	case "smtp":
		return w.SMTP.put(n)
	case "database":
		return w.Database.put(n)
	case "redis":
		return w.Redis.put(n)
	case "store":
		var cfg storeCfg

		if err := cfg.put(n); err != nil {
			return err
		}

		if w.Stores == nil {
			w.Stores = make(map[string]storeCfg)
		}
		w.Stores[n.label] = cfg
	case "provider":
		var cfg providerCfg

		if w.Providers == nil {
			w.Providers = make(map[string]providerCfg)
		}
		if err := cfg.put(n); err != nil {
			return err
		}
		w.Providers[n.label] = cfg
	default:
		return n.err("unknown configuration parameter: " + n.name)
	}
	return nil
}

func (w *Worker) Pidfile() *os.File             { return w.pidfile }
func (w *Worker) Parallelism() int              { return w.parallelism }
func (w *Worker) Driver() string                { return w.driver }
func (w *Worker) Queue() string                 { return w.queue }
func (w *Worker) Consumer() *curlyq.Consumer    { return w.consumer }
func (w *Worker) Timeout() time.Duration        { return w.timeout }
func (w *Worker) DB() *sqlx.DB                  { return w.db }
func (w *Worker) Redis() *redis.Client          { return w.redis }
func (w *Worker) SMTP() (*mail.Client, string)  { return w.smtp, w.postmaster }
func (w *Worker) Artifacts() fs.Store           { return w.artifacts }
func (w *Worker) Objects() fs.Store             { return w.objects }
func (w *Worker) BlockCipher() *crypto.Block    { return w.block }
func (w *Worker) Log() *log.Logger              { return w.log }
func (w *Worker) Providers() *provider.Registry { return w.providers }
