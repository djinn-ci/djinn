package config

import (
	"io"
	"os"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/log"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/mcmathja/curlyq"

	"github.com/pelletier/go-toml"
)

type schedulerCfg struct {
	Pidfile string

	Drivers    []string
	ShareQueue bool `toml:"share_queue"`

	Database databaseCfg
	Redis    redisCfg

	Log logCfg
}

type Scheduler struct {
	pidfile *os.File

	db    *sqlx.DB
	redis *redis.Client

	log *log.Logger

	producers map[string]*curlyq.Producer
}

func decodeScheduler(r io.Reader) (schedulerCfg, error) {
	var cfg schedulerCfg

	if err := toml.NewDecoder(r).Decode(&cfg); err != nil {
		return cfg, errors.Err(err)
	}

	if cfg.Log.Level == "" {
		cfg.Log.Level = "INFO"
	}

	if cfg.Log.File == "" {
		cfg.Log.File = "/dev/stdout"
	}
	return cfg, errors.Err(cfg.validate())
}

func DecodeScheduler(r io.Reader) (Scheduler, error) {
	var s Scheduler

	cfg, err := decodeScheduler(r)

	if err != nil {
		return s, errors.Err(err)
	}

	s.pidfile, err = mkpidfile(cfg.Pidfile)

	if err != nil {
		return s, errors.Err(err)
	}

	s.log = log.New(os.Stdout)
	s.log.SetLevel(cfg.Log.Level)

	if cfg.Log.File != "/dev/stdout" {
		f, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

		if err != nil {
			return s, errors.Err(err)
		}
		s.log.SetWriter(f)
	}

	s.log.Info.Println("logging initiliazed, writing to", cfg.Log.File)

	s.db, err = connectdb(s.log, cfg.Database)

	if err != nil {
		return s, errors.Err(err)
	}

	s.redis, err = connectredis(s.log, cfg.Redis)

	if err != nil {
		return s, errors.Err(err)
	}

	s.producers = make(map[string]*curlyq.Producer)

	for _, driver := range cfg.Drivers {
		queue := defaultBuildQueue + "_" + driver

		if cfg.ShareQueue {
			queue = defaultBuildQueue
		}

		s.producers[driver] = curlyq.NewProducer(&curlyq.ProducerOpts{
			Client: s.redis,
			Queue:  queue,
			Logger: serverLogger{log: s.log},
		})
	}
	return s, nil
}

func (cfg schedulerCfg) validate() error {
	if err := cfg.Database.validate(); err != nil {
		return err
	}

	if cfg.Redis.Addr == "" {
		return errors.New("missing redis address")
	}

	if len(cfg.Drivers) == 0 {
		return errors.New("no build drivers configured")
	}
	return nil
}

func (s Scheduler) Pidfile() *os.File { return s.pidfile }

func (s Scheduler) DB() *sqlx.DB { return s.db }

func (s Scheduler) Redis() *redis.Client { return s.redis }

func (s Scheduler) Log() *log.Logger { return s.log }

func (s Scheduler) Producers() map[string]*curlyq.Producer { return s.producers }
