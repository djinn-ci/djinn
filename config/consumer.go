package config

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"

	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/log"

	"github.com/mcmathja/curlyq"

	"github.com/jmoiron/sqlx"

	"github.com/go-redis/redis"
)

type consumerCfg struct {
	Pidfile     string
	Queue       string
	Attempts    int
	Parallelism int

	Log logCfg

	Database databaseCfg
	Redis    redisCfg

	Stores map[string]storeCfg
}

type Consumer struct {
	pidfile *os.File

	queue       string
	parallelism int

	log *log.Logger

	consumer *curlyq.Consumer

	db    *sqlx.DB
	redis *redis.Client

	stores map[string]fs.Store
}

func DecodeConsumer(name string, r io.Reader) (*Consumer, error) {
	errh := func(name string, line, col int, msg string) {
		fmt.Fprintf(os.Stderr, "%s,%d:%d - %s\n", name, line, col, msg)
	}

	p := newParser(name, r, errh)

	nodes := p.parse()

	if err := p.err(); err != nil {
		return nil, err
	}

	var cfg0 consumerCfg

	for _, n := range nodes {
		if err := cfg0.put(n); err != nil {
			return nil, err
		}
	}

	var err error

	cfg := &Consumer{}
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

	if cfg0.Attempts == 0 {
		cfg0.Attempts = 5
	}

	cfg.queue = cfg0.Queue
	cfg.parallelism = cfg0.Parallelism

	if cfg.parallelism == 0 {
		cfg.parallelism = int(runtime.NumCPU())
	}

	cfg.db, err = connectdb(cfg.log, cfg0.Database)

	if err != nil {
		return nil, err
	}

	cfg.redis, err = connectredis(cfg.log, cfg0.Redis)

	if err != nil {
		return nil, err
	}

	cfg.consumer = curlyq.NewConsumer(&curlyq.ConsumerOpts{
		Queue:                cfg0.Queue,
		Client:               cfg.redis,
		Logger:               workerLogger{log: cfg.log},
		ProcessorConcurrency: cfg.parallelism,
		JobMaxAttempts:       cfg0.Attempts,
	})

	for name, storecfg := range cfg0.Stores {
		if _, ok := blockstores[storecfg.Type]; !ok {
			return nil, errors.New("unknown store type: "+storecfg.Type)
		}
		cfg.stores[name] = blockstores[storecfg.Type](storecfg.Path, storecfg.Limit)
	}
	return cfg, nil
}

func (c *consumerCfg) put(n *node) error {
	switch n.name {
	case "pidfile":
		if n.lit != stringLit {
			return n.err("pidfile must be a string")
		}
		c.Pidfile = n.value
	case "queue":
		if n.lit != stringLit {
			return n.err("queue must be a string")
		}
		c.Queue = n.value
	case "attempts":
		if n.lit != numberLit {
			return n.err("attempts must be an integer")
		}

		i, err := strconv.ParseInt(n.value, 10, 64)

		if err != nil {
			return n.err("parallelism is not a valid integer")
		}
		c.Attempts = int(i)
	case "parallelism":
		if n.lit != numberLit {
			return n.err("parallelism must be an integer")
		}

		i, err := strconv.ParseInt(n.value, 10, 64)

		if err != nil {
			return n.err("parallelism is not a valid integer")
		}
		c.Parallelism = int(i)
	case "log":
		return c.Log.put(n)
	case "redis":
		return c.Redis.put(n)
	case "database":
		return c.Database.put(n)
	case "store":
		var cfg storeCfg

		if err := cfg.put(n); err != nil {
			return err
		}

		if c.Stores == nil {
			c.Stores = make(map[string]storeCfg)
		}
		c.Stores[n.label] = cfg
	default:
		return n.err("unknown configuration parameter: " + n.name)
	}
	return nil
}

func (c *Consumer) Pidfile() *os.File          { return c.pidfile }
func (c *Consumer) QueueName() string          { return c.queue }
func (c *Consumer) Consumer() *curlyq.Consumer { return c.consumer }
func (c *Consumer) DB() *sqlx.DB               { return c.db }
func (c *Consumer) Redis() *redis.Client       { return c.redis }
func (c *Consumer) Log() *log.Logger           { return c.log }

func (c *Consumer) Store(name string) (fs.Store, bool) {
	store, ok := c.stores[name]
	return store, ok
}
