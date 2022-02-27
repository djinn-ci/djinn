package config

import (
	"io"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/log"

	"github.com/andrewpillar/config"
)

type curatorCfg struct {
	Pidfile string

	Log map[string]string

	Database databaseCfg

	Store map[string]storeCfg
}

type Curator struct {
	pidfile string

	log *log.Logger

	db database.Pool

	artifacts fs.Store
}

func (c *Curator) Pidfile() string     { return c.pidfile }
func (c *Curator) DB() database.Pool   { return c.db }
func (c *Curator) Artifacts() fs.Store { return c.artifacts }
func (c *Curator) Log() *log.Logger    { return c.log }

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
