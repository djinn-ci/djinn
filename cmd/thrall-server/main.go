package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/server"

	"github.com/go-redis/redis"
)

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

	srv := server.Server{
		HttpAddr:  cfg.Net.Listen,
		HttpsAddr: cfg.Net.SSL.Listen,
		SSLCert:   cfg.Net.SSL.Cert,
		SSLKey:    cfg.Net.SSL.Key,
	}

	uiSrv := uiServer{
		Server: srv,
		store:  model.NewStore(db),
		client: client,
		hash:   []byte(cfg.Crypto.Hash),
		key:    []byte(cfg.Crypto.Key),
		assets: cfg.Assets,
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
