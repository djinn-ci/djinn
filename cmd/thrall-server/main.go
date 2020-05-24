package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/andrewpillar/cli"

	buildweb "github.com/andrewpillar/thrall/build/web"
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	imageweb "github.com/andrewpillar/thrall/image/web"
	keyweb "github.com/andrewpillar/thrall/key/web"
	"github.com/andrewpillar/thrall/log"
	namespaceweb "github.com/andrewpillar/thrall/namespace/web"
	"github.com/andrewpillar/thrall/oauth2"
	oauth2web "github.com/andrewpillar/thrall/oauth2/web"
	objectweb "github.com/andrewpillar/thrall/object/web"
	"github.com/andrewpillar/thrall/provider"
	repoweb "github.com/andrewpillar/thrall/repo/web"
	"github.com/andrewpillar/thrall/server"
	"github.com/andrewpillar/thrall/session"
	"github.com/andrewpillar/thrall/user"
	variableweb "github.com/andrewpillar/thrall/variable/web"
	userweb "github.com/andrewpillar/thrall/user/web"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"

	goredis "github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/RichardKnop/machinery/v1"
	qconfig "github.com/RichardKnop/machinery/v1/config"

	_ "github.com/lib/pq"
)

var (
	Version string
	Build   string
)

func connectDB(cfg config.Database) *sqlx.DB {
	host, port, err := net.SplitHostPort(cfg.Addr)

	if err != nil {
		log.Error.Fatal(err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		host,
		port,
		cfg.Name,
		cfg.Username,
		cfg.Password,
	)

	log.Debug.Println("opening postgresql connection with:", dsn)

	db, err := sqlx.Open("postgres", dsn)

	if err != nil {
		log.Error.Fatal(err)
	}

	log.Debug.Println("testing connection to database")

	if err := db.Ping(); err != nil {
		log.Error.Fatal(err)
	}
	return db
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

	db := connectDB(cfg.Database)

	log.Info.Println("connected to postgresql database")

	redis := goredis.NewClient(&goredis.Options{
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

	queues := make(map[string]*machinery.Server)

	for _, d := range cfg.Drivers {
		queue, err := machinery.NewServer(&qconfig.Config{
			Broker:        broker,
			DefaultQueue:  d.Queue,
			ResultBackend: broker,
		})

		if err != nil {
			log.Error.Fatalf("failed to setup queue %s: %s\n", d.Queue, err)
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

	var imageStore filestore.FileStore

	if cfg.Images.Path != "" {
		imageStore, err = filestore.New(cfg.Images)

		if err != nil {
			log.Error.Fatalf("failed to create image store: %s\n", errors.Cause(err))
		}
	}

	objectStore, err := filestore.New(cfg.Objects)

	if err != nil {
		log.Error.Fatalf("failed to create object store: %s\n", errors.Cause(err))
	}

	artifacts, err := filestore.New(cfg.Artifacts)

	if err != nil {
		log.Error.Fatalf("failed to create artifact store: %s\n", errors.Cause(err))
	}

	authKey := []byte(cfg.Crypto.Auth)

	if len(authKey) != 32 {
		log.Error.Fatalf("auth key must be 32 bytes in size\n")
	}

	providers := make(map[string]oauth2.Provider)

	for _, p := range cfg.Providers {
		provider, err := provider.New(p.Name, provider.Opts{
			Host:         cfg.Host,
			Endpoint:     p.Endpoint,
			Secret:       p.Secret,
			ClientID:     p.ClientID,
			ClientSecret: p.ClientSecret,
		})

		if err != nil {
			log.Error.Fatalf("failed to configure oauth provider: %s\n", errors.Cause(err))
		}
		providers[p.Name] = provider
	}

	handler := web.Handler{
		DB:           db,
		Store:        session.New(redis, blockKey),
		SecureCookie: securecookie.New(hashKey, blockKey),
		Users:        user.NewStore(db),
		Tokens:       oauth2.NewTokenStore(db),
	}

	middleware := web.Middleware{Handler: handler}

	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			web.JSONError(w, "Not found", http.StatusNotFound)
			return
		}
		web.HTMLError(w, "Not found", http.StatusNotFound)
	})

	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			web.JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		web.HTMLError(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	srv := server.Server{
		Server: &http.Server{
			Addr: cfg.Net.Listen,
		},
		Router: r,
		Cert:   cfg.Net.SSL.Cert,
		Key:    cfg.Net.SSL.Key,
	}

	serveUI := cmd.Flags.IsSet("ui")
	serveAPI := cmd.Flags.IsSet("api")

	// No flags were given, so serve both.
	if !serveUI && !serveAPI {
		serveUI = true
		serveAPI = true
	}

	srv.AddRouter("auth", &userweb.Router{
		Providers:  providers,
		Middleware: middleware,
	})

	srv.AddRouter("build", &buildweb.Router{
		Middleware: middleware,
		Artifacts:  artifacts,
		Redis:      redis,
		Queues:     queues,
		Providers:  providers,
	})

	srv.AddRouter("namespace", &namespaceweb.Router{
		Middleware: middleware,
	})

	srv.AddRouter("repo", &repoweb.Router{
		Redis:      redis,
		Providers:  providers,
		Middleware: middleware,
	})

	srv.AddRouter("image", &imageweb.Router{
		Middleware: middleware,
		FileStore:  imageStore,
		Limit:      cfg.Images.Limit,
	})

	srv.AddRouter("object", &objectweb.Router{
		Middleware: middleware,
		FileStore:  objectStore,
		Limit:      cfg.Objects.Limit,
	})

	srv.AddRouter("variable", &variableweb.Router{
		Middleware: middleware,
	})

	srv.AddRouter("key", &keyweb.Router{
		Middleware: middleware,
	})

	srv.AddRouter("oauth2", &oauth2web.Router{
		Middleware: middleware,
		Providers:  providers,
	})

	srv.Init(handler)

	if serveUI {
		ui := server.UI{
			Server: srv,
			CSRF:   csrf.Protect(
				authKey,
				csrf.RequestHeader("X-CSRF-Token"),
				csrf.FieldName("csrf_token"),
			),
		}

		ui.Init()
		ui.Register("auth")
		ui.Register("build", buildweb.Gate(db))
		ui.Register("repo", repoweb.Gate(db))
		ui.Register("namespace", namespaceweb.Gate(db))
		ui.Register("image", imageweb.Gate(db))
		ui.Register("object", objectweb.Gate(db))
		ui.Register("variable", variableweb.Gate(db))
		ui.Register("key", keyweb.Gate(db))
		ui.Register("oauth2")
	}

	var apiPrefix string

	if serveAPI {
		if serveUI {
			apiPrefix = "/api"
		}

		api := server.API{
			Server: srv,
			Prefix: apiPrefix,
		}

		api.Init()
		api.Register("build", buildweb.Gate(db))
//		api.Register("namespace", namespaceweb.Gate(db))
//		api.Register("image", imageweb.Gate(db))
//		api.Register("object", objectweb.Gate(db))
//		api.Register("variable", variableweb.Gate(db))
//		api.Register("key", keyweb.Gate(db))
	}

	go func() {
		if err := srv.Serve(); err != nil {
			if cause := errors.Cause(err); cause != http.ErrServerClosed {
				log.Error.Fatal(cause)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second * 15))
	defer cancel()

	log.Info.Println("thrall-server started on", cfg.Net.Listen)

	if apiPrefix != "" {
		log.Info.Println("api routes being served under", cfg.Net.Listen+apiPrefix)
	}

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	sig := <-c

	srv.Shutdown(ctx)

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
			fmt.Println("thrall-server", Version, Build)
		},
	})

	cmd.AddFlag(&cli.Flag{
		Name: "ui",
		Long: "--ui",
	})

	cmd.AddFlag(&cli.Flag{
		Name: "api",
		Long: "--api",
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
