package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/server"
	"github.com/andrewpillar/thrall/session"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/securecookie"

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

	crypto.Key = []byte(cfg.Crypto.Block)

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

	broker := "redis://"

	if cfg.Redis.Password != "" {
		broker += cfg.Redis.Password + "@"
	}

	broker += cfg.Redis.Addr

	queues := make(map[string]*machinery.Server)

	for _, d := range cfg.Drivers {
		queue, err := machinery.NewServer(&qconfig.Config{
			Broker:        broker,
			DefaultQueue:  d.Queue,
			ResultBackend: broker,
		})

		if err != nil {
			log.Error.Fatalf("failed to setup queue: %s\n", err)
		}

		queues[d.Type] = queue
	}

	hashKey := []byte(cfg.Crypto.Hash)
	blockKey := []byte(cfg.Crypto.Block)

	if len(hashKey) < 32 || len(hashKey) > 64 {
		log.Error.Fatalf("hash key is either too long or too short, make sure it between 32 and 64 bytes in size\n")
	}

	if len(blockKey) != 16 && len(blockKey) != 24 && len(blockKey) != 32 {
		log.Error.Fatalf("block key must be either 16, 24, or 32 bytes in size\n")
	}

	handler := web.Handler{
		Store:        session.New(client, blockKey),
		SecureCookie: securecookie.New(hashKey, blockKey),
		Users:        model.UserStore{
			Store: model.Store{
				DB: db,
			},
		},
	}

	middleware := web.Middleware{
		Handler: handler,
	}

	var (
		images     filestore.FileStore
		imageLimit int64
	)

	if cfg.Images != "" {
		images, err = filestore.New(cfg.Images)

		if err != nil {
			log.Error.Fatalf("failed to create image store: %s\n", err)
		}

		u, _ := url.Parse(cfg.Images)

		imageLimit, _ = strconv.ParseInt(u.Query().Get("limit"), 10, 64)
	}

	objects, err := filestore.New(cfg.Objects)

	if err != nil {
		log.Error.Fatalf("failed to create object store: %s\n", err)
	}

	objectUrl, _ := url.Parse(cfg.Objects)

	objectLimit, _ := strconv.ParseInt(objectUrl.Query().Get("limit"), 10, 64)

	artifacts, err := filestore.New(cfg.Artifacts)

	if err != nil {
		log.Error.Fatalf("failed to create artifact store: %s\n", err)
	}

	authKey := []byte(cfg.Crypto.Auth)

	if len(authKey) != 32 {
		log.Error.Fatalf("auth key must be 32 bytes in size\n")
	}

	providers := make(map[string]oauth2.Provider)

	for _, p := range cfg.Providers {
		provider, err := oauth2.NewProvider(p.Name, p.ClientID, p.ClientSecret, cfg.Host, p.Secret, p.Endpoint)

		if err != nil {
			log.Error.Fatalf("failed to configure oauth provider: %s\n", errors.Cause(err))
		}

		providers[p.Name] = provider
	}

	srv := server.Server{
		Server: &http.Server{
			Addr: cfg.Net.Listen,
		},
		Cert:        cfg.Net.SSL.Cert,
		Key:         cfg.Net.SSL.Key,
		DB:          db,
		Redis:       client,
		CSRFToken:   authKey,
		Images:      images,
		Artifacts:   artifacts,
		Objects:     objects,
		Queues:      queues,
		Providers:   providers,
		ImageLimit:  imageLimit,
		ObjectLimit: objectLimit,
		Handler:     handler,
		Middleware:  middleware,
	}

	ui := server.UI{
		Server: srv,
		Assets: "public",
	}

	ui.Init()

	ui.Hook()

	ui.Auth()
	ui.Oauth()
	ui.Guest()

	ui.Namespace()
	ui.Build()

	if cfg.Images != "" {
		ui.Image()
	}

	ui.Object()
	ui.Variable()
	ui.Key()

	go func() {
		if err := ui.Serve(); err != nil {
			cause := errors.Cause(err)

			if cause != http.ErrServerClosed {
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

	ui.Shutdown(ctx)

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
