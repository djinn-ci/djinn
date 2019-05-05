package main

import (
	"fmt"
	"os"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/collector"
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/placer"
	"github.com/andrewpillar/thrall/server"
)

func mainCommand(c cli.Command) {
	f, err := os.Open(c.Flags.GetString("config"))

	if err != nil {
		log.Error.Fatalf("failed to open worker config: %s\n", err)
	}

	defer f.Close()

	cfg, err := config.DecodeWorker(f)

	if err != nil {
		log.Error.Fatalf("failed to decode worker config: %s\n", err)
	}

	log.SetLevel(cfg.Log.Level)

	logf, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		log.Error.Fatalf("failed to open log file %s: %s\n", cfg.Log.File, err)
	}

	defer logf.Close()

	log.SetLogger(log.NewStdLog(logf))

	db, err := model.Connect(
		cfg.Database.Addr,
		cfg.Database.Name,
		cfg.Database.Username,
		cfg.Database.Password,
	)

	if err != nil {
		log.Error.Fatalf("failed to establish postgresql connection: %s\n", err)
	}

	log.Info.Println("connected to postgresql database")

	srv := server.Server{
		HttpAddr:  cfg.Net.Listen,
		HttpsAddr: cfg.Net.SSL.Listen,
		SSLCert:   cfg.Net.SSL.Cert,
		SSLKey:    cfg.Net.SSL.Key,
	}

	w := worker{
		Server:        srv,
		concurrency:   cfg.Parallelism,
		redisAddr:     cfg.Redis.Addr,
		redisPassword: cfg.Redis.Password,
		store:         model.NewStore(db),
		placer:        placer.NewFileSystem("."),
		collector:     collector.NewFileSystem("."),
	}

	if err := w.init(); err != nil {
		log.Error.Fatalf("failed to initialize worker: %s\n", errors.Cause(err))
	}

	if err := w.serve(); err != nil {
		log.Error.Fatalf("failed to launch worker: %s\n", errors.Cause(err))
	}
}

func main() {
	c := cli.New()

	cmd := c.MainCommand(mainCommand)

	cmd.AddFlag(&cli.Flag{
		Name:     "config",
		Short:    "-c",
		Long:     "--config",
		Argument: true,
		Default:  "thrall-worker.toml",
	})

	if err := c.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}