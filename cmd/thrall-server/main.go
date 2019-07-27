package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/server"

	"github.com/go-redis/redis"

	"github.com/RichardKnop/machinery/v1"
	qconfig "github.com/RichardKnop/machinery/v1/config"
)

var Build string

func mainCommand(cmd cli.Command) {
	f, err := os.Open(cmd.Flags.GetString("config"))

	if err != nil {
		log.Error.Fatalf("failed to open server config: %s\n", err)
	}

	defer f.Close()

	cfg, err := config.DecodeServer(f)

	if err != nil {
		log.Error.Fatalf("failed to decode server config: %s\n", err)
	}

	log.SetLevel(cfg.Log.Level)

	logf, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		log.Error.Fatalf("failed to open log file %s: %s\n", cfg.Log.File, err)
	}

	defer logf.Close()

	log.SetLogger(log.NewStdLog(logf))

	if err := crypto.InitHashing(cfg.Crypto.Salt, 8); err != nil {
		log.Error.Fatalf("failed to initialize hashing mechanism: %s\n", err)
	}

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

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})

	if _, err := client.Ping().Result(); err != nil {
		log.Error.Fatalf("failed to ping redis: %s\n", err)
	}

	log.Info.Println("connected to redis database")

	srv := &server.Server{
		HttpAddr:  cfg.Net.Listen,
		HttpsAddr: cfg.Net.SSL.Listen,
		SSLCert:   cfg.Net.SSL.Cert,
		SSLKey:    cfg.Net.SSL.Key,
		CSRFToken: []byte(cfg.Crypto.Auth),
	}

	broker := "redis://"

	if cfg.Redis.Password != "" {
		broker += cfg.Redis.Password + "@"
	}

	broker += cfg.Redis.Addr

	if len(cfg.Drivers) == 1 && cfg.Drivers[0] == "*" {
		cfg.Drivers = driver.All
	}

	for _, d := range cfg.Drivers {
		qcfg := &qconfig.Config{
			Broker:        broker,
			DefaultQueue:  "thrall_builds_" + d,
			ResultBackend: broker,
		}

		qsrv, err := machinery.NewServer(qcfg)

		if err != nil {
			log.Error.Fatalf("failed to create queue server: %s\n", err)
		}

		log.Debug.Println("adding build queue:", qcfg.DefaultQueue)

		srv.AddQueue(d, qsrv)
	}

	artifacts, err := filestore.New(cfg.Artifacts)

	if err != nil {
		log.Error.Fatalf("failed to create artifact store: %s\n", err)
	}

	objects, err := filestore.New(cfg.Objects)

	if err != nil {
		log.Error.Fatalf("failed to create object store: %s\n", err)
	}

	uiSrv := uiServer{
		Server:    srv,
		db:        db,
		client:    client,
		limit:     cfg.Objects.Limit,
		artifacts: artifacts,
		objects:   objects,
		hash:      []byte(cfg.Crypto.Hash),
		key:       []byte(cfg.Crypto.Key),
		assets:    "public",
	}

	uiSrv.init()
	uiSrv.Serve()

	log.Info.Println("thrall-server started")

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	sig := <-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second * 15))
	defer cancel()

	uiSrv.Shutdown(ctx)

	log.Info.Println("signal:", sig, "received, shutting down")
}

func main() {
	c := cli.New()

	cmd := c.MainCommand(mainCommand)

	c.AddFlag(&cli.Flag{
		Name:      "version",
		Long:      "--version",
		Exclusive: true,
		Handler:   func(f cli.Flag, c cli.Command) {
			fmt.Println("thrall-server", Build)
		},
	})

	cmd.AddFlag(&cli.Flag{
		Name:     "config",
		Short:    "-c",
		Long:     "--config",
		Argument: true,
		Default:  "thrall-server.toml",
	})

	if err := c.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
