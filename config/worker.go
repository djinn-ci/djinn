package config

import (
	"io"
	"net/smtp"
	"os"
	"time"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/driver"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/log"
	"github.com/andrewpillar/djinn/provider"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/pelletier/go-toml"

	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
)

type Worker struct {
	pidfile *os.File

	parallelism int
	timeout     time.Duration

	block *crypto.Block

	db         *sqlx.DB
	redis      *redis.Client
	smtp       *smtp.Client
	postmaster string

	artifacts block.Store
	objects   block.Store

	log *log.Logger

	queue *machinery.Server

	drivers   *driver.Registry
	providers *provider.Registry
}

type workerCfg struct {
	Pidfile     string
	Parallelism int
	Queue       string
	Timeout     string

	Crypto cryptoCfg

	SMTP smtpCfg

	Database databaseCfg
	Redis    redisCfg

	Images    storageCfg
	Artifacts storageCfg
	Objects   storageCfg

	Log logCfg

	Providers []providerCfg
}

func decodeWorker(r io.Reader) (workerCfg, error) {
	var cfg workerCfg

	if err := toml.NewDecoder(r).Decode(&cfg); err != nil {
		return cfg, errors.Err(err)
	}

	if cfg.Timeout == "" {
		cfg.Timeout = "30m"
	}

	if cfg.Images.Type == "" {
		cfg.Images.Type = "file"
	}

	if cfg.Artifacts.Type == "" {
		cfg.Artifacts.Type = "file"
	}

	if cfg.Objects.Type == "" {
		cfg.Objects.Type = "file"
	}

	if cfg.Log.Level == "" {
		cfg.Log.Level = "INFO"
	}

	if cfg.Log.File == "" {
		cfg.Log.File = "/dev/stdout"
	}
	return cfg, errors.Err(cfg.validate())
}

func DecodeWorker(r io.Reader) (Worker, error) {
	var w Worker

	cfg, err := decodeWorker(r)

	if err != nil {
		return w, errors.Err(err)
	}

	w.pidfile, err = mkpidfile(cfg.Pidfile)

	if err != nil {
		return w, errors.Err(err)
	}

	w.log = log.New(os.Stdout)
	w.log.SetLevel(cfg.Log.Level)

	if cfg.Log.File != "/dev/stdout" {
		f, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

		if err != nil {
			return w, errors.Err(err)
		}
		w.log.SetWriter(f)
	}

	w.log.Info.Println("logging initiliazed, writing to", cfg.Log.File)

	w.timeout, err = time.ParseDuration(cfg.Timeout)

	if err != nil {
		return w, errors.Err(err)
	}

	w.block, err = crypto.NewBlock([]byte(cfg.Crypto.Block))

	if err != nil {
		return w, errors.Err(err)
	}

	w.db, err = connectdb(w.log, cfg.Database)

	if err != nil {
		return w, errors.Err(err)
	}

	w.redis, err = connectredis(w.log, cfg.Redis)

	if err != nil {
		return w, errors.Err(err)
	}

	broker := "redis://"

	if cfg.Redis.Password != "" {
		broker += cfg.Redis.Password + "@"
	}

	broker += cfg.Redis.Addr

	w.queue, err = machinery.NewServer(&config.Config{
		Broker:        broker,
		DefaultQueue:  cfg.Queue,
		ResultBackend: broker,
	})

	if err != nil {
		return w, err
	}

	w.smtp, err = connectsmtp(w.log, cfg.SMTP)

	if err != nil {
		return w, errors.Err(err)
	}

	w.postmaster = cfg.SMTP.Admin

	w.objects = blockstores[cfg.Objects.Type](cfg.Objects.Path, cfg.Objects.Limit)
	w.artifacts = blockstores[cfg.Artifacts.Type](cfg.Artifacts.Path, cfg.Artifacts.Limit)

	if err := w.objects.Init(); err != nil {
		return w, err
	}

	if err := w.artifacts.Init(); err != nil {
		return w, err
	}

	w.providers = provider.NewRegistry()

	for _, p := range cfg.Providers {
		fn, ok := providerFactories[p.Name]

		if !ok {
			return w, errors.New("unknown provider: " + p.Name)
		}
		w.providers.Register(p.Name, fn("", p.Endpoint, p.Secret, p.ClientID, p.ClientSecret))
	}
	return w, nil
}

func (cfg workerCfg) validate() error {
	if cfg.Queue == "" {
		return errors.New("missing queue name")
	}

	if len(cfg.Crypto.Block) != 16 && len(cfg.Crypto.Block) != 24 && len(cfg.Crypto.Block) != 32 {
		return errors.New("invalid block key, must be either 16, 24, or 32 bytes in length")
	}

	if err := cfg.Database.validate(); err != nil {
		return err
	}

	if cfg.Redis.Addr == "" {
		return errors.New("missing redis address")
	}

	if err := cfg.SMTP.validate(); err != nil {
		return err
	}

	if cfg.Artifacts.Path == "" {
		return errors.New("missing artifacts storage path")
	}

	if cfg.Objects.Path == "" {
		return errors.New("missing objects storage path")
	}
	return nil
}

func (w Worker) Pidfile() *os.File { return w.pidfile }

func (w Worker) Timeout() time.Duration { return w.timeout }

func (w Worker) DB() *sqlx.DB { return w.db }

func (w Worker) Redis() *redis.Client { return w.redis }

func (w Worker) SMTP() (*smtp.Client, string) { return w.smtp, w.postmaster }

func (w Worker) Artifacts() block.Store { return w.artifacts }

func (w Worker) Objects() block.Store { return w.objects }

func (w Worker) BlockCipher() *crypto.Block { return w.block }

func (w Worker) Log() *log.Logger { return w.log }

func (w Worker) Queue() *machinery.Server { return w.queue }

func (w Worker) Parallelism() int { return w.parallelism }

func (w Worker) Providers() *provider.Registry { return w.providers }

func (w Worker) Drivers() *driver.Registry { return w.drivers }
