package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/log"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"
)

var (
	blockstores = map[string]func(string, int64) block.Store{
		"file": func(dsn string, limit int64) block.Store {
			return block.NewFilesystemWithLimit(dsn, limit)
		},
	}

	Build   string
	Version string
)

func curateArtifacts(log *log.Logger, limit int64, users *user.Store, artifacts *build.ArtifactStore, store block.Store) error {
	uu, err := users.All(query.Where("cleanup", "=", true))

	if err != nil {
		return errors.Err(err)
	}

	mm := database.ModelSlice(len(uu), user.Model(uu))

	aa, err := artifacts.All(
		query.Where("size", ">", 0),
		query.Where("user_id", "IN", database.MapKey("id", mm)...),
		query.OrderAsc("created_at"),
	)

	if err != nil {
		return errors.Err(err)
	}

	sums := make(map[int64]int64)
	curated := make(map[int64][]string)

	for _, a := range aa {
		sum := sums[a.UserID]
		sum += a.Size.Int64

		if sum >= limit {
			curated[a.UserID] = append(curated[a.UserID], a.Hash)
		}
	}

	for userId, hashes := range curated {
		log.Debug.Println("curated", len(hashes), "artifacts for user", userId)

		for _, hash := range hashes {
			log.Debug.Println("removing artifact with hash", hash)

			if err := store.Remove(hash); err != nil {
				log.Error.Println("failed to remove artfiact", hash, errors.Err(err))
			}
		}
	}
	return nil
}

func run(stdout, stderr io.Writer, args []string) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	var (
		configfile  string
		limit       int64
		showversion bool
	)

	flags.BoolVar(&showversion, "version", false, "show the version and exit")
	flags.Int64Var(&limit, "limit", 1 << 30, "remove artifacts that go over this limit")
	flags.StringVar(&configfile, "config", "djinn-curator.toml", "the config file to use")
	flags.Parse(args[1:])

	if showversion {
		fmt.Fprintf(stdout, "%s %s %s\n", args[0], Version, Build)
		return nil
	}

	log := log.New(stdout)

	f, err := os.Open(configfile)

	if err != nil {
		return err
	}

	defer f.Close()

	cfg, err := config.DecodeCurator(f)

	if err != nil {
		return err
	}

	if cfg.Pidfile != "" {
		pidf, err := os.OpenFile(cfg.Pidfile, os.O_WRONLY|os.O_CREATE, 0660)

		if err != nil {
			return err
		}

		pidf.Write([]byte(strconv.FormatInt(int64(os.Getpid()), 10)))
		pidf.Close()
	}

	log.SetLevel(cfg.Log.Level)

	logf, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		return err
	}

	defer logf.Close()

	log.Info.Println("logging initialized, writing to", logf.Name())

	log.SetWriter(logf)

	host, port, err := net.SplitHostPort(cfg.Database.Addr)

	if err != nil {
		return err
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		host,
		port,
		cfg.Database.Name,
		cfg.Database.Username,
		cfg.Database.Password,
	)

	log.Debug.Println("connecting to postgresql database with:", dsn)

	db, err := database.Connect(dsn)

	if err != nil {
		return err
	}

	defer db.Close()

	log.Info.Println("connected to postgresql database")

	store := blockstores[cfg.Artifacts.Type](cfg.Artifacts.Path, cfg.Artifacts.Limit)

	if err := store.Init(); err != nil {
		return err
	}

	users := user.NewStore(db)
	artifacts := build.NewArtifactStore(db)

	t := time.NewTicker(time.Minute)
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

loop:
	for {
		select {
		case <-t.C:
			if err := curateArtifacts(log, limit, users, artifacts, store); err != nil {
				log.Error.Println("failed to curate artifacts", errors.Err(err))
			}
		case sig := <-c:
			log.Info.Println("signal:", sig, "received, shutting down")
			break loop
		}
	}

	if cfg.Pidfile != "" {
		if err := os.RemoveAll(cfg.Pidfile); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := run(os.Stdout, os.Stderr, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
