package config

import (
	"io"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/log"

	"github.com/andrewpillar/config"

	"github.com/go-redis/redis"

	"github.com/mcmathja/curlyq"
)

type schedCfg struct {
	Pidfile string

	Interval  time.Duration
	BatchSize int64 `config:"batch_size"`

	Log map[string]string

	Drivers []string

	Crypto cryptoCfg

	Database databaseCfg
	Redis    redisCfg
}

type Scheduler struct {
	pidfile      string
	interval     time.Duration
	batchsize    int64
	hasher       *crypto.Hasher
	db           *database.Pool
	redis        *redis.Client
	log          *log.Logger
	driverQueues map[string]*curlyq.Producer
}

func (s *Scheduler) Pidfile() string                           { return s.pidfile }
func (s *Scheduler) Interval() time.Duration                   { return s.interval }
func (s *Scheduler) BatchSize() int64                          { return s.batchsize }
func (s *Scheduler) Hasher() *crypto.Hasher                    { return s.hasher }
func (s *Scheduler) DB() *database.Pool                        { return s.db }
func (s *Scheduler) Redis() *redis.Client                      { return s.redis }
func (s *Scheduler) Log() *log.Logger                          { return s.log }
func (s *Scheduler) DriverQueues() map[string]*curlyq.Producer { return s.driverQueues }

func DecodeScheduler(name string, r io.Reader) (*Scheduler, error) {
	var cfg schedCfg

	dec := config.NewDecoder(name, decodeOpts...)

	if err := dec.Decode(&cfg, r); err != nil {
		return nil, err
	}

	sched := &Scheduler{}

	var err error

	sched.pidfile, err = mkpidfile(cfg.Pidfile)

	if err != nil {
		return nil, err
	}

	sched.log, err = logger(cfg.Log)

	if err != nil {
		return nil, err
	}

	if cfg.BatchSize == 0 {
		cfg.BatchSize = 1000
	}

	sched.log.Info.Println("batch size set to", cfg.BatchSize)

	sched.interval = cfg.Interval
	sched.batchsize = cfg.BatchSize

	sched.hasher, err = cfg.Crypto.hasher()

	if err != nil {
		return nil, err
	}

	sched.db, err = cfg.Database.connect(sched.log)

	if err != nil {
		return nil, err
	}

	sched.redis, err = cfg.Redis.connect(sched.log)

	if err != nil {
		return nil, err
	}

	sched.driverQueues = driverQueues(sched.log, sched.redis, cfg.Drivers)

	return sched, nil
}
