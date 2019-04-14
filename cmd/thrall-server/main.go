package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/session"

	"github.com/gorilla/securecookie"

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

	var httpsServer *http.Server

	hash := []byte(cfg.Crypto.Hash)
	key := []byte(cfg.Crypto.Key)

	if err := model.Connect(cfg.Database.Addr, cfg.Database.Name, cfg.Database.Username, cfg.Database.Password); err != nil {
		log.Error.Fatalf("failed to establish postgresql connection: %s\n", err)
	}

	log.Info.Println("connected to postgresql database")

	sc := securecookie.New(hash, key)

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})

	if err != nil {
		log.Error.Fatalf("failed to establish redis connection: %s\n", err)
	}

	if _, err := client.Ping().Result(); err != nil {
		log.Error.Fatalf("failed to ping redis: %s\n", err)
	}

	log.Info.Println("connected to redis database")

	store := session.New(client, key)

	var handler http.Handler = registerWebRoutes(web.New(sc, store), cfg.Assets)

	if cfg.Log.Access {
		handler = web.NewLog(handler)
	}

	handler = web.NewSpoof(handler)

	httpServer := &http.Server{
		Addr:         cfg.Net.Listen,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      handler,
	}

	if cfg.Net.SSL.Cert != "" && cfg.Net.SSL.Key != "" {
		httpsServer = &http.Server{
			Addr:         cfg.Net.SSL.Listen,
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
			Handler:      handler,
		}

		httpServer.Handler = web.NewSpoof(web.NewSecureRedirect(cfg.Net.SSL.Listen, handler))

		go func() {
			if err := httpsServer.ListenAndServeTLS(cfg.Net.SSL.Cert, cfg.Net.SSL.Key); err != nil {
				log.Error.Println("error serving request:", err)
			}
		}()
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Error.Println("error serving request:", err)
		}
	}()

	log.Info.Println("thrall-server started")

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	sig := <-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second * 15))
	defer cancel()

	httpServer.Shutdown(ctx)

	if httpsServer != nil {
		httpsServer.Shutdown(ctx)
	}

	log.Info.Println(sig, "received, shutting down")
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
