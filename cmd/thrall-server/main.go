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

	"github.com/gorilla/securecookie"

	"gopkg.in/boj/redistore.v1"
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
		log.Error.Fatalf("failed to establish database connection: %s\n", err)
	}

	sc := securecookie.New(hash, key)

	store, err := redistore.NewRediStore(cfg.Redis.Idle, "tcp", cfg.Redis.Addr, cfg.Redis.Password, key)

	if err != nil {
		log.Error.Fatalf("failed to create session store: %s\n", err)
	}

	router := registerWebRoutes(web.New(sc, store), cfg.Assets)

	weblog := web.NewLog(router)
	spoof := web.NewSpoof(weblog)

	httpServer := &http.Server{
		Addr:         cfg.Net.Listen,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      spoof,
	}

	if cfg.Net.SSL.Cert != "" && cfg.Net.SSL.Key != "" {
		httpsServer = &http.Server{
			Addr:         cfg.Net.SSL.Listen,
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
			Handler:      spoof,
		}

		httpServer.Handler = web.NewSpoof(web.NewSecureRedirect(cfg.Net.SSL.Listen, router))

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

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second * 15))
	defer cancel()

	httpServer.Shutdown(ctx)

	if httpsServer != nil {
		httpsServer.Shutdown(ctx)
	}

	log.Info.Println("shutting down")
}

func main() {
	c := cli.New()

	cmd := c.MainCommand(mainCommand)

	cmd.AddFlag(&cli.Flag{
		Name:     "config",
		Short:    "-c",
		Long:     "--config",
		Argument: true,
		Default:  "thrall-server.yml",
	})

	if err := c.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
