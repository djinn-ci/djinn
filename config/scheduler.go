package config

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/log"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/mcmathja/curlyq"
)

type schedulerCfg struct {
	Pidfile string

	Interval  time.Duration
	BatchSize int64

	Log logCfg

	Drivers []string

	Crypto    Crypto
	Database databaseCfg
	Redis    redisCfg
}

type Scheduler struct {
	pidfile *os.File

	interval  time.Duration
	batchsize int64

	hasher *crypto.Hasher
	db     *sqlx.DB
	redis  *redis.Client

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
		f, err := os.OpenFile(cfg0.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

		if err != nil {
			return nil, err
		}
		cfg.log.SetWriter(f)
	}

	cfg.log.Info.Println("logging initialized, writing to", cfg0.Log.File)

	if cfg0.Interval == 0 {
		cfg0.Interval = time.Minute
	}

	cfg.log.Info.Println("batch interval set to", cfg0.Interval)

	if cfg0.BatchSize == 0 {
		cfg0.BatchSize = 1000
	}

	cfg.log.Info.Println("batch size set to", cfg0.BatchSize)

	cfg.interval = cfg0.Interval
	cfg.batchsize = cfg0.BatchSize

	cfg.hasher = &crypto.Hasher{
		Salt:   string(cfg0.Crypto.Salt),
		Length: 8,
	}

	if err := cfg.hasher.Init(); err != nil {
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

	cfg.producers = make(map[string]*curlyq.Producer)

	for _, driver := range cfg0.Drivers {
		queue := defaultBuildQueue + "_" + driver

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
	case "interval":
		if n.lit != stringLit {
			return n.err("interval must be a duration string")
		}

		d, err := time.ParseDuration(n.value)

		if err != nil {
			return n.err("interval is not a valid duration string")
		}
		s.Interval = d
	case "batchsize":
		if n.lit != numberLit {
			return n.err("batchsize must be an integer")
		}

		i, err := strconv.ParseInt(n.value, 10, 64)

		if err != nil {
			return n.err("batchsize must be an integer")
		}
		s.BatchSize = i
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
	case "crypto":
		// These values are not needed by the worker itself, but necessary to
		// pass validation, so spoof for now.
		s.Crypto.Block = []byte("00000000000000000000000000000000")
		s.Crypto.Auth = []byte("00000000000000000000000000000000")
		s.Crypto.Hash = []byte("00000000000000000000000000000000")

		if err := s.Crypto.put(n); err != nil {
			return err
		}

		s.Crypto.Block = nil
		s.Crypto.Auth = nil
		s.Crypto.Hash = nil
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
func (s *Scheduler) Interval() time.Duration { return s.interval }
func (s *Scheduler) BatchSize() int64 { return s.batchsize }
func (s *Scheduler) Hasher() *crypto.Hasher { return s.hasher }
func (s *Scheduler) DB() *sqlx.DB { return s.db }
func (s *Scheduler) Redis() *redis.Client { return s.redis }
func (s *Scheduler) Log() *log.Logger { return s.log }
func (s *Scheduler) Producers() map[string]*curlyq.Producer { return s.producers }
