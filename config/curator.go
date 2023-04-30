package config

import (
	"io"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/log"

	"github.com/andrewpillar/config"
	"github.com/andrewpillar/fs"
)

type curatorCfg struct {
	Pidfile string

	Log map[string]string

	Interval time.Duration

	Database databaseCfg

	Store map[string]storeCfg
}

type Curator struct {
	pidfile   string
	log       *log.Logger
	interval  time.Duration
	db        *database.Pool
	artifacts fs.FS
}

func (c *Curator) Pidfile() string         { return c.pidfile }
func (c *Curator) DB() *database.Pool      { return c.db }
func (c *Curator) Artifacts() fs.FS        { return c.artifacts }
func (c *Curator) Log() *log.Logger        { return c.log }
func (c *Curator) Interval() time.Duration { return c.interval }

func DecodeCurator(name string, r io.Reader) (*Curator, error) {
	var cfg curatorCfg

	dec := config.NewDecoder(name, decodeOpts...)

	if err := dec.Decode(&cfg, r); err != nil {
		return nil, err
	}

	curator := &Curator{}

	var err error

	curator.pidfile, err = mkpidfile(cfg.Pidfile)

	if err != nil {
		return nil, err
	}

	curator.log, err = logger(cfg.Log)

	if err != nil {
		return nil, err
	}

	if cfg.Interval == 0 {
		cfg.Interval = time.Minute * 5
	}

	curator.interval = cfg.Interval

	curator.db, err = cfg.Database.connect(curator.log)

	if err != nil {
		return nil, err
	}

	s, ok := cfg.Store["artifacts"]

	if !ok {
		return nil, errors.New("artifacts store not configured")
	}

	curator.artifacts, err = s.store()

	if err != nil {
		return nil, err
	}
	return curator, nil
}
