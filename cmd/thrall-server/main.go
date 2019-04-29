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

	"github.com/go-redis/redis"
)

type server struct {
	store  *model.Store
	client *redis.Client

	httpAddr  string
	httpsAddr string
	sslCert   string
	sslKey    string

	hash []byte
	key  []byte

	http  *http.Server
	https *http.Server
}

func (s *server) init(h http.Handler) {
	s.http = &http.Server{
		Addr:         s.httpAddr,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      h,
	}

	if s.httpsAddr != "" && s.sslCert != "" && s.sslKey != "" {
		s.https = &http.Server{
			Addr:         s.httpsAddr,
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
			Handler:      h,
		}

		s.http.Handler = web.NewSecureRedirect(s.https.Addr)
	}
}

func (s server) serve() {
	if s.https != nil {
		go func() {
			if err := s.https.ListenAndServeTLS(s.sslCert, s.sslKey); err != nil {
				log.Error.Println("error serving request:", err)
			}
		}()
	}

	go func() {
		if err := s.http.ListenAndServe(); err != nil {
			log.Error.Println("error serving request:", err)
		}
	}()

	log.Info.Println("thrall-server started")

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	sig := <-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second * 15))
	defer cancel()

	s.http.Shutdown(ctx)

	if s.https != nil {
		s.https.Shutdown(ctx)
	}

	log.Info.Println("signal:", sig, "received, shutting down")
}

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

	if err != nil {
		log.Error.Fatalf("failed to establish redis connection: %s\n", err)
	}

	if _, err := client.Ping().Result(); err != nil {
		log.Error.Fatalf("failed to ping redis: %s\n", err)
	}

	log.Info.Println("connected to redis database")

	srv := server{
		store:     model.NewStore(db),
		client:    client,
		httpAddr:  cfg.Net.Listen,
		httpsAddr: cfg.Net.SSL.Listen,
		sslCert:   cfg.Net.SSL.Cert,
		sslKey:    cfg.Net.SSL.Key,
		hash:      []byte(cfg.Crypto.Hash),
		key:       []byte(cfg.Crypto.Key),
	}

	uiSrv := uiServer{
		server: srv,
		assets: cfg.Assets,
	}

	uiSrv.init()
	uiSrv.serve()
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
