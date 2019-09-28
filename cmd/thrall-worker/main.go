package main

import (
	"fmt"
	nethttp "net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/http"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"

	"github.com/RichardKnop/machinery/v1"
	qconfig "github.com/RichardKnop/machinery/v1/config"
)

var Build string

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

	if len(cfg.Drivers) == 0 {
		log.Error.Fatalf("no drivers configured, exiting\n")
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

	srv := &http.Server{
		Addr: cfg.Net.Listen,
		Cert: cfg.Net.SSL.Cert,
		Key:  cfg.Net.SSL.Key,
	}

	artifacts, err := filestore.New(cfg.Artifacts)

	if err != nil {
		log.Error.Fatalf("failed to create artifact store: %s\n", err)
	}

	objects, err := filestore.New(cfg.Objects)

	if err != nil {
		log.Error.Fatalf("failed to create object store: %s\n", err)
	}

	duration, err := time.ParseDuration(cfg.Timeout)

	if err != nil {
		log.Error.Fatalf("failed to parse timeout duration: %s\n", err)
	}

	store := model.Store{
		DB: db,
	}

	if len(cfg.Drivers) == 1 && cfg.Drivers[0] == "*" {
		cfg.Drivers = driver.All
	}

	// Sort drivers so the final queue name will be the same regardless of
	// order in the config file.
	sort.Strings(cfg.Drivers)

	broker := "redis://"

	if cfg.Redis.Password != "" {
		broker += cfg.Redis.Password
	}

	broker += cfg.Redis.Addr

	qname := []string{"thrall", "builds"}
	qname = append(qname, cfg.Drivers...)

	queue, err := machinery.NewServer(&qconfig.Config{
		Broker:        broker,
		DefaultQueue:  strings.Join(qname, "_"),
		ResultBackend: broker,
	})

	if err != nil {
		log.Error.Fatalf("failed to create queue server: %s\n", err)
	}

	w := worker{
		Server:        srv,
		queue:         queue,
		concurrency:   cfg.Parallelism,
		driverCfg:     config.Driver{
			SSH:  cfg.SSH,
			Qemu: cfg.Qemu,
		},
		timeout:       duration,
		store:         store,
		objects:       objects,
		artifacts:     artifacts,
	}

	w.init(strings.Join(qname, "_"))

	go func() {
		if err := w.Serve(); err != nil {
			cause := errors.Cause(err)

			if cause != nethttp.ErrServerClosed {
				log.Error.Fatal(cause)
			}
		}
	}()

	if err := w.worker.Launch(); err != nil {
		log.Error.Fatalf("failed to launch worker: %s\n", errors.Cause(err))
	}
}

func main() {
	c := cli.New()

	cmd := c.MainCommand(mainCommand)

	c.AddFlag(&cli.Flag{
		Name:      "version",
		Long:      "--version",
		Exclusive: true,
		Handler:   func(f cli.Flag, c cli.Command) {
			fmt.Println("thrall-worker", Build)
		},
	})

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
