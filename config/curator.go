package config

import (
	"fmt"
	"io"
	"os"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/fs"
	"github.com/andrewpillar/djinn/log"

	"github.com/jmoiron/sqlx"
)

type curatorCfg struct {
	Pidfile string

	Log logCfg

	Database  databaseCfg
	Stores    map[string]storeCfg
}

type Curator struct {
	pidfile *os.File

	log *log.Logger

	db *sqlx.DB

	artifacts fs.Store
}

func DecodeCurator(name string, r io.Reader) (*Curator, error) {
	errh := func(name string, line, col int, msg string) {
		fmt.Fprintf(os.Stderr, "%s,%d:%d - %s\n", name, line, col, msg)
	}

	p := newParser(name, r, errh)

	nodes := p.parse()

	if err := p.err(); err != nil {
		return nil, err
	}

	var cfg0 curatorCfg

	for _, n := range nodes {
		if err := cfg0.put(n); err != nil {
			return nil, err
		}
	}

	var err error

	cfg := &Curator{}
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

	store, ok := cfg0.Stores["artifacts"]

	if !ok {
		return nil, errors.New("artifact store not configured")
	}

	cfg.artifacts = blockstores[store.Type](store.Path, store.Limit)

	if err := cfg.artifacts.Init(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *curatorCfg) put(n *node) error {
	switch n.name {
	case "pidfile":
		if n.lit != stringLit {
			return n.err("pidfile must be a string")
		}
		c.Pidfile = n.value
	case "log":
		return c.Log.put(n)
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

func (c *Curator) Pidfile() *os.File { return c.pidfile }
func (c *Curator) DB() *sqlx.DB { return c.db }
func (c *Curator) Artifacts() fs.Store { return c.artifacts }
func (c *Curator) Log() *log.Logger { return c.log }
