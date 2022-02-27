package config

import (
	"io"
	"runtime"

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/log"

	"github.com/andrewpillar/config"

	"github.com/mcmathja/curlyq"

	"github.com/go-redis/redis"
)

type consumerCfg struct {
	Pidfile string

	Attempts    int
	Parallelism int

	Log map[string]string

	Crypto cryptoCfg

	Database databaseCfg
	Redis    redisCfg

	Store map[string]storeCfg
}

type Consumer struct {
	pidfile string

	parallelism int

	log *log.Logger

	aesgcm *crypto.AESGCM

	consumer *curlyq.ConsumerOpts

	db    database.Pool
	redis *redis.Client

	images fs.Store
}

func (c *Consumer) Pidfile() string                    { return c.pidfile }
func (c *Consumer) ConsumerOpts() *curlyq.ConsumerOpts { return c.consumer }
func (c *Consumer) DB() database.Pool                  { return c.db }
func (c *Consumer) Redis() *redis.Client               { return c.redis }
func (c *Consumer) Log() *log.Logger                   { return c.log }
func (c *Consumer) AESGCM() *crypto.AESGCM             { return c.aesgcm }
func (c *Consumer) Images() fs.Store                   { return c.images }

func DecodeConsumer(name string, r io.Reader) (*Consumer, error) {
	var cfg consumerCfg

	dec := config.NewDecoder(name, decodeOpts...)

	if err := dec.Decode(&cfg, r); err != nil {
		return nil, err
	}

	consumer := &Consumer{}

	var err error

	consumer.pidfile, err = mkpidfile(cfg.Pidfile)

	if err != nil {
		return nil, errors.Err(err)
	}

	consumer.aesgcm, err = cfg.Crypto.aesgcm()

	if err != nil {
		return nil, errors.Err(err)
	}

	consumer.log, err = logger(cfg.Log)

	if err != nil {
		return nil, errors.Err(err)
	}

	if cfg.Attempts == 0 {
		cfg.Attempts = 5
	}

	consumer.parallelism = cfg.Parallelism

	if consumer.parallelism == 0 {
		consumer.parallelism = int(runtime.NumCPU())
	}

	consumer.db, err = cfg.Database.connect(consumer.log)

	if err != nil {
		return nil, err
	}

	consumer.redis, err = cfg.Redis.connect(consumer.log)

	if err != nil {
		return nil, err
	}

	s, ok := cfg.Store["images"]

	if !ok {
		return nil, errors.New("images store not configured")
	}

	consumer.images, err = s.store()

	if err != nil {
		return nil, err
	}

	consumer.consumer = &curlyq.ConsumerOpts{
		Client: consumer.redis,
		Logger: log.Queue{
			Logger: consumer.log,
		},
		ProcessorConcurrency: consumer.parallelism,
		JobMaxAttempts:       cfg.Attempts,
	}
	return consumer, nil
}
