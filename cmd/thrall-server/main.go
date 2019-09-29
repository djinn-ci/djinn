package main

import (
	"context"
	"fmt"
	nethttp "net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/http"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"

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

	crypto.Key = []byte(cfg.Crypto.Key)

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

	defer client.Close()

	log.Info.Println("connected to redis database")

	srv := server{
		Server: &http.Server{
			Addr:      cfg.Net.Listen,
			Cert:      cfg.Net.SSL.Cert,
			Key:       cfg.Net.SSL.Key,
			CSRFToken: []byte(cfg.Crypto.Auth),
		},
		client: client,
	}

	broker := "redis://"

	if cfg.Redis.Password != "" {
		broker += cfg.Redis.Password + "@"
	}

	broker += cfg.Redis.Addr

	if len(cfg.Drivers) == 1 && cfg.Drivers[0] == "*" {
		cfg.Drivers = driver.All
	}

	// Sort drivers so the final queue name will be the same regardless of
	// order in the config file.
	sort.Strings(cfg.Drivers)

	srv.drivers = make(map[string]struct{})

	for _, d := range cfg.Drivers {
		srv.drivers[d] = struct{}{}
	}

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

	artifacts, err := filestore.New(cfg.Artifacts)

	if err != nil {
		log.Error.Fatalf("failed to create artifact store: %s\n", err)
	}

	objects, err := filestore.New(cfg.Objects)

	if err != nil {
		log.Error.Fatalf("failed to create object store: %s\n", err)
	}

	srv.db = db
	srv.client = client
	srv.objects = objects
	srv.artifacts = artifacts
	srv.hash = []byte(cfg.Crypto.Hash)
	srv.key = []byte(cfg.Crypto.Key)
	srv.limit = cfg.Objects.Limit
	srv.queue = queue

	uiSrv := uiServer{
		server: srv,
		assets: "public",
	}

	uiSrv.init()

	go func() {
		if err := uiSrv.Serve(); err != nil {
			cause := errors.Cause(err)

			if cause != nethttp.ErrServerClosed {
				log.Error.Fatal(cause)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second * 15))
	defer cancel()

	log.Info.Println("thrall-server started on", cfg.Net.Listen)

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, os.Kill)

	sig := <-c

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
