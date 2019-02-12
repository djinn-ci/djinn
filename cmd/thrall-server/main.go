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
	"github.com/andrewpillar/thrall/handler"
	"github.com/andrewpillar/thrall/log"

	"github.com/gorilla/securecookie"

	"gopkg.in/boj/redistore.v1"
)

func mainCommand(c cli.Command) {
	f, err := os.Open(c.Flags.GetString("config"))

	if err != nil {
		log.Error.Fatalf("failed to open server config: %s\n", err)
	}

	cfg, err := config.DecodeServer(f)

	if err != nil {
		log.Error.Fatalf("failed to decode server config: %s\n", err)
	}

	log.SetLevel(cfg.Log.Level)

	lf, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		log.Error.Fatalf("failed to open log file %s: %s\n", cfg.Log.File, err)
	}

	defer lf.Close()

	log.SetLogger(log.NewStdLog(lf))

	var httpsServer *http.Server

	hash := []byte(cfg.Crypto.Hash)
	key := []byte(cfg.Crypto.Key)

	sc := securecookie.New(hash, key)

	store, err := redistore.NewRediStore(cfg.Redis.Idle, "tcp", cfg.Redis.Addr, cfg.Redis.Password, key)

	if err != nil {
		log.Error.Fatalf("failed to create redis store: %s\n", err)
	}

	baseHandler := handler.New(sc, store)

	httpServer := &http.Server{
		Addr:         cfg.Net.Listen,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      http.HandlerFunc(baseHandler.Home),
	}

	if cfg.Net.SSL.Cert != "" && cfg.Net.SSL.Key != "" {
		httpsServer = &http.Server{
			Addr:         cfg.Net.SSL.Listen,
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
			Handler:      http.HandlerFunc(baseHandler.Home),
		}

		httpServer.Handler = handler.NewSecureRedirect(cfg.Net.SSL.Listen, http.HandlerFunc(baseHandler.Home))

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

	sig := make(chan os.Signal, 1)

	signal.Notify(sig, os.Interrupt)

	<-sig

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

	cmd := c.Main(mainCommand)

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
