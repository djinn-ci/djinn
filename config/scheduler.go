package config

import (
	"fmt"
	"io"
	"os"

	"github.com/andrewpillar/djinn/log"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/mcmathja/curlyq"
)

type schedulerCfg struct {
	Pidfile string

	Log logCfg

	Drivers    []string
	ShareQueue bool

	Database databaseCfg
	Redis    redisCfg
}

type Scheduler struct {
	pidfile *os.File

	db    *sqlx.DB
	redis *redis.Client

	log *log.Logger

	producers map[string]*curlyq.Producer
}

func DecodeScheduler(name string, r io.Reader) (*Scheduler, error) {
	errh := func(name string, line, col int, msg string) {
		fmt.Fprintf(os.Stderr, "%s,%d:%d - %s\n", name, line, col, msg)
	}

	p := newParser(name, r, errh)

	nodes := p.parse()

	if err := p.err(); err != nil {
		return nil, err
	}

	var cfg0 schedulerCfg

	for _, n := range nodes {
		if err := cfg0.put(n); err != nil {
			return nil, err
		}
	}

	var err error

	cfg := &Scheduler{}
	cfg.pidfile, err = mkpidfile(cfg0.Pidfile)

	if err != nil {
		return nil, err
	}

	cfg.log = log.New(os.Stdout)
	cfg.log.SetLevel(cfg0.Log.Level)

	if cfg0.Log.File != "/dev/stdout" {
		f, err := os.OpenFile(cfg0.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)

		if err != nil {
			return nil, err
		}
		cfg.log.SetWriter(f)
	}

	cfg.log.Info.Println("logging initialized, writing to", cfg0.Log.File)

	cfg.db, err = connectdb(cfg.log, cfg0.Database)

	if err != nil {
		return nil, err
	}

	cfg.redis, err = connectredis(cfg.log, cfg0.Redis)

	if err != nil {
		return nil, err
	}

	cfg.producers = make(map[string]*curlyq.Producer)

	for _, driver := range cfg0.Drivers {
		queue := defaultBuildQueue + "_" + driver

		if cfg0.ShareQueue {
			queue = defaultBuildQueue
		}

		cfg.producers[driver] = curlyq.NewProducer(&curlyq.ProducerOpts{
			Client: cfg.redis,
			Queue:  queue,
			Logger: serverLogger{log: cfg.log},
		})
	}
	return cfg, nil
}

func (s *schedulerCfg) put(n *node) error {
	switch n.name {
	case "pidfile":
		if n.lit != stringLit {
			return n.err("pidfile must be a string")
		}
		s.Pidfile = n.value
	case "log":
		return s.Log.put(n)
	case "drivers":
		if n.list == nil {
			return n.err("drivers must be an array")
		}

		var walkerr error

		n.list.walk(func(n *node) {
			if n.lit != stringLit {
				walkerr = n.err("drivers must be an array of strings")
				return
			}
			s.Drivers = append(s.Drivers, n.value)
		})

		if walkerr != nil {
			return walkerr
		}
	case "share_queue":
		if n.lit != boolLit {
			return n.err("share_queue must be a boolean")
		}

		if n.value == "true" {
			s.ShareQueue = true
			return nil
		}
		s.ShareQueue = false
	case "database":
		return s.Database.put(n)
	case "redis":
		return s.Redis.put(n)
	default:
		return n.err("unknown configuration parameter: " + n.name)
	}
	return nil
}

func (s *Scheduler) Pidfile() *os.File { return s.pidfile }
func (s *Scheduler) DB() *sqlx.DB { return s.db }
func (s *Scheduler) Redis() *redis.Client { return s.redis }
func (s *Scheduler) Log() *log.Logger { return s.log }
func (s *Scheduler) Producers() map[string]*curlyq.Producer { return s.producers }
