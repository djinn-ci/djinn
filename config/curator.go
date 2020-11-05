package config

import (
	"io"
	"os"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/log"

	"github.com/jmoiron/sqlx"

	"github.com/pelletier/go-toml"
)

type curatorCfg struct {
	Pidfile string

	Database  databaseCfg
	Artifacts storageCfg

	Log logCfg
}

type Curator struct {
	pidfile *os.File

	db *sqlx.DB

	artifacts block.Store

	log *log.Logger
}

func decodeCurator(r io.Reader) (curatorCfg, error) {
	var cfg curatorCfg

	if err := toml.NewDecoder(r).Decode(&cfg); err != nil {
		return cfg, errors.Err(err)
	}

	if cfg.Artifacts.Type == "" {
		cfg.Artifacts.Type = "file"
	}

	if cfg.Log.Level == "" {
		cfg.Log.Level = "INFO"
	}

	if cfg.Log.File == "" {
		cfg.Log.File = "/dev/stdout"
	}
	return cfg, errors.Err(cfg.validate())
}

func DecodeCurator(r io.Reader) (Curator, error) {
	var c Curator

	cfg, err := decodeCurator(r)

	if err != nil {
		return c, errors.Err(err)
	}

	c.pidfile, err = mkpidfile(cfg.Pidfile)

	if err != nil {
		return c, errors.Err(err)
	}

	c.log = log.New(os.Stdout)
	c.log.SetLevel(cfg.Log.Level)

	if cfg.Log.File != "/dev/stdout" {
		f, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

		if err != nil {
			return c, errors.Err(err)
		}
		c.log.SetWriter(f)
	}

	c.log.Info.Println("logging initiliazed, writing to", cfg.Log.File)

	c.db, err = connectdb(c.log, cfg.Database)

	if err != nil {
		return c, errors.Err(err)
	}

	c.artifacts = blockstores[cfg.Artifacts.Type](cfg.Artifacts.Path, 0)

	if c.artifacts.Init(); err != nil {
		return c, errors.Err(err)
	}
	return c, nil
}

func (cfg curatorCfg) validate() error {
	if err := cfg.Database.validate(); err != nil {
		return err
	}

	if cfg.Artifacts.Path == "" {
		return errors.New("missing artifacts storage path")
	}
	return nil
}

func (c Curator) Pidfile() *os.File { return c.pidfile }

func (c Curator) DB() *sqlx.DB { return c.db }

func (c Curator) Artifacts() block.Store { return c.artifacts }

func (c Curator) Log() *log.Logger { return c.log }
