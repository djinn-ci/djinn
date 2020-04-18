package main

import (
	"fmt"
	"os"
	"time"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/log"

	"github.com/andrewpillar/cli"

	"github.com/go-redis/redis"

	"github.com/RichardKnop/machinery/v1"
	qconfig "github.com/RichardKnop/machinery/v1/config"
)

var (
	Version string
	Build   string
)

func mainCommand(c cli.Command) {
	cf, err := os.Open(c.Flags.GetString("config"))

	if err != nil {
		log.Error.Fatalf("failed to open worker config: %s\n", err)
	}

	defer cf.Close()

	df, err := os.Open(c.Flags.GetString("driver"))

	if err != nil {
		log.Error.Fatalf("failed to open driver config: %s\n", err)
	}

	defer df.Close()

	cfg, err := config.DecodeWorker(cf)

	if err != nil {
		log.Error.Fatalf("failed to decode worker config: %s\n", err)
	}

	driverCfg, err := config.DecodeDriver(df)

	if err != nil {
		log.Error.Fatalf("failed to decode driver config: %s\n", err)
	}

	if cfg.Queue == "" {
		log.Error.Fatalf("no queue to work from\n")
	}

	log.SetLevel(cfg.Log.Level)

	logf, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		log.Error.Fatalf("failed to open log file %s: %s\n", cfg.Log.File, err)
	}

	defer logf.Close()

	crypto.Key = []byte(cfg.Crypto.Block)

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

	redis := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})

	if _, err := redis.Ping().Result(); err != nil {
		log.Error.Fatalf("failed to ping redis: %s\n", err)
	}

	defer redis.Close()

	log.Info.Println("connected to redis database")

	broker := "redis://"

	if cfg.Redis.Password != "" {
		broker += cfg.Redis.Password + "@"
	}
	broker += cfg.Redis.Addr

	queue, err := machinery.NewServer(&qconfig.Config{
		Broker:        broker,
		DefaultQueue:  cfg.Queue,
		ResultBackend: broker,
	})

	if err != nil {
		log.Error.Fatalf("failed to setup queue %s: %s\n", cfg.Queue, err)
	}

	objects, err := filestore.New(cfg.Objects)

	if err != nil {
		log.Error.Fatalf("failed to create object store: %s\n", errors.Cause(err))
	}

	artifacts, err := filestore.New(cfg.Artifacts)

	if err != nil {
		log.Error.Fatalf("failed to create artifact store: %s\n", errors.Cause(err))
	}

	if _, err := filestore.New(cfg.Images); err != nil {
		log.Error.Fatalf("failed to create image store: %s\n", errors.Cause(err))
	}

	timeout, err := time.ParseDuration(cfg.Timeout)

	if err != nil {
		log.Error.Fatalf("failed to parse worker timeout: %s\n", err)
	}

	w := worker{
		db:        db,
		redis:     redis,
		driver:    driverCfg,
		timeout:   timeout,
		server:    queue,
		placer:    objects,
		collector: artifacts,
	}

	w.init(cfg.Queue, cfg.Parallelism)

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
				fmt.Println("thrall-worker", Version, Build)
		},
	})

	cmd.AddFlag(&cli.Flag{
		Name:     "config",
		Short:    "-c",
		Long:     "--config",
		Argument: true,
		Default:  "thrall-worker.toml",
	})

	cmd.AddFlag(&cli.Flag{
		Name:     "driver",
		Short:    "-d",
		Long:     "--driver",
		Argument: true,
		Default:  "thrall-driver.toml",
	})

	if err := c.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
