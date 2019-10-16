package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"

	"github.com/go-redis/redis"

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

	if cfg.Queue == "" {
		log.Error.Fatalf("no queue to work from\n", err)
	}

	log.SetLevel(cfg.Log.Level)

	logf, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		log.Error.Fatalf("failed to open log file %s: %s\n", cfg.Log.File, err)
	}

	defer logf.Close()

	crypto.Key = []byte(cfg.Crypto.Key)

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

	broker := "redis://"

	if cfg.Redis.Password != "" {
		broker += cfg.Redis.Password
	}

	broker += cfg.Redis.Addr

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})

	if _, err := client.Ping().Result(); err != nil {
		log.Error.Fatalf("failed to ping redis: %s\n", err)
	}

	queue, err := machinery.NewServer(&qconfig.Config{
		Broker:        broker,
		DefaultQueue:  cfg.Queue,
		ResultBackend: broker,
	})

	if err != nil {
		log.Error.Fatalf("failed to create queue server: %s\n", err)
	}

	if _, err := filestore.New(cfg.Images); err != nil {
		cause := errors.Cause(err)

		log.Error.Fatalf("failed to open images location: %s\n", cause)
	}

	cfg.Qemu.Disks = filepath.Join(cfg.Images, "qemu")

	w := worker{
		client:      client,
		queue:       queue,
		concurrency: cfg.Parallelism,
		driverCfg:   config.Driver{
			SSH:  cfg.SSH,
			Qemu: cfg.Qemu,
		},
		timeout:       duration,
		store:         store,
		objects:       objects,
		artifacts:     artifacts,
	}

	w.init(cfg.Queue)

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
