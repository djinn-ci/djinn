package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/block"
	buildweb "github.com/andrewpillar/thrall/build/web"
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	imageweb "github.com/andrewpillar/thrall/image/web"
	keyweb "github.com/andrewpillar/thrall/key/web"
	"github.com/andrewpillar/thrall/log"
	namespaceweb "github.com/andrewpillar/thrall/namespace/web"
	"github.com/andrewpillar/thrall/oauth2"
	oauth2web "github.com/andrewpillar/thrall/oauth2/web"
	objectweb "github.com/andrewpillar/thrall/object/web"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/provider/github"
	"github.com/andrewpillar/thrall/provider/gitlab"
	providerweb "github.com/andrewpillar/thrall/provider/web"
	"github.com/andrewpillar/thrall/server"
	"github.com/andrewpillar/thrall/session"
	"github.com/andrewpillar/thrall/user"
	userweb "github.com/andrewpillar/thrall/user/web"
	variableweb "github.com/andrewpillar/thrall/variable/web"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"

	"github.com/go-redis/redis"

	"github.com/RichardKnop/machinery/v1"
	qconfig "github.com/RichardKnop/machinery/v1/config"

	_ "github.com/lib/pq"
)

var (
	Version string
	Build   string

	blockstores = map[string]func(string, int64) block.Store{
		"file": func(dsn string, limit int64) block.Store {
			return block.NewFilesystemWithLimit(dsn, limit)
		},
	}

	providerFactories = map[string]provider.Factory{
		"github": func(host, endpoint, secret, clientId, clientSecret string) provider.Interface {
			return github.New(host, endpoint, secret, clientId, clientSecret)
		},
		"gitlab": func(host, endpoint, secret, clientId, clientSecret string) provider.Interface {
			return gitlab.New(host, endpoint, secret, clientId, clientSecret)
		},
	}
)

func main() {
	var (
		showversion bool
		serveui     bool
		serveapi    bool
		configfile  string
	)

	flag.BoolVar(&showversion, "version", false, "show the version and exit")
	flag.BoolVar(&serveui, "ui", false, "serve only the ui endpoints")
	flag.BoolVar(&serveapi, "api", false, "serve only the api endpoints")
	flag.StringVar(&configfile, "config", "djinn-server.toml", "the config file to use")
	flag.Parse()

	if showversion {
		fmt.Println(os.Args[0], Version, Build)
		os.Exit(0)
	}

	if !serveui && !serveapi {
		serveui = true
		serveapi = true
	}

	log := log.New(os.Stdout)

	f, err := os.Open(configfile)

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

	if cfg.Pidfile != "" {
		pidf, err := os.OpenFile(cfg.Pidfile, os.O_WRONLY|os.O_CREATE, 0660)

		if err != nil {
			log.Error.Fatalf("failed to create pidfile: %s\n", err)
		}

		pidf.Write([]byte(strconv.FormatInt(int64(os.Getpid()), 10)))
		pidf.Close()
	}

	log.SetLevel(cfg.Log.Level)

	logf, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		log.Error.Fatalf("failed to open log file %s: %s\n", cfg.Log.File, err)
	}

	defer logf.Close()

	log.Info.Println("logging initialized, writing to", logf.Name())

	log.SetWriter(logf)

	blockCipher, err := crypto.NewBlock([]byte(cfg.Crypto.Block))

	if err != nil {
		log.Error.Fatalf("failed to setup block cipher: %s\n", errors.Cause(err))
	}

	hasher := &crypto.Hasher{
		Salt:   cfg.Crypto.Salt,
		Length: 8,
	}

	if err := hasher.Init(); err != nil {
		log.Error.Fatalf("failed to initialize hashing mechanism: %s\n", err)
	}

	host, port, err := net.SplitHostPort(cfg.Database.Addr)

	if err != nil {
		log.Error.Fatal(err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		host,
		port,
		cfg.Database.Name,
		cfg.Database.Username,
		cfg.Database.Password,
	)

	log.Debug.Println("connecting to postgresql database with:", dsn)

	db, err := database.Connect(dsn)

	if err != nil {
		log.Error.Fatalf("failed to connect to database: %s\n", errors.Cause(err))
	}

	log.Info.Println("connected to postgresql database")

	redis := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})

	log.Debug.Println("connecting to redis database with:", cfg.Redis.Addr, cfg.Redis.Password)

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

	imageStore := blockstores[cfg.Images.Type](cfg.Images.Path, cfg.Images.Limit)
	objectStore := blockstores[cfg.Objects.Type](cfg.Objects.Path, cfg.Objects.Limit)
	artifactStore := blockstores[cfg.Artifacts.Type](cfg.Artifacts.Path, cfg.Artifacts.Limit)

	if err := imageStore.Init(); err != nil {
		log.Error.Fatalf("failed to initialize image store: %s\n", errors.Cause(err))
	}

	if err := objectStore.Init(); err != nil {
		log.Error.Fatalf("failed to initialize object store: %s\n", errors.Cause(err))
	}

	if err := artifactStore.Init(); err != nil {
		log.Error.Fatalf("failed to initialize artifact store: %s\n", errors.Cause(err))
	}

	authKey := []byte(cfg.Crypto.Auth)

	if len(authKey) != 32 {
		log.Error.Fatalf("auth key must be 32 bytes in size\n")
	}

	providers := provider.NewRegistry()

	for _, p := range cfg.Providers {
		factory, ok := providerFactories[p.Name]

		if !ok {
			log.Error.Fatalf("unknown provider: %s\n", p.Name)
		}
		providers.Register(
			p.Name,
			factory(cfg.Host, p.Endpoint, p.Secret, p.ClientID, p.ClientSecret),
		)
	}

	handler := web.Handler{
		DB:           db,
		Log:          log,
		Store:        session.New(redis, blockKey),
		SecureCookie: securecookie.New(hashKey, blockKey),
		Users:        user.NewStore(db),
	}

	middleware := web.Middleware{
		Handler: handler,
		Tokens:  oauth2.NewTokenStore(db),
	}

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
		Log:    log,
		Router: r,
		Cert:   cfg.Net.SSL.Cert,
		Key:    cfg.Net.SSL.Key,
	}

	srv.AddRouter("auth", &userweb.Router{
		Registry:   providers,
		Middleware: middleware,
	})

	srv.AddRouter("build", &buildweb.Router{
		Block:      blockCipher,
		Middleware: middleware,
		Artifacts:  artifactStore,
		Redis:      redis,
		Hasher:     hasher,
		Queues:     queues,
		Registry:   providers,
	})

	srv.AddRouter("namespace", &namespaceweb.Router{
		Middleware: middleware,
	})

	srv.AddRouter("provider", &providerweb.Router{
		Redis:      redis,
		Block:      blockCipher,
		Registry:   providers,
		Middleware: middleware,
	})

	srv.AddRouter("image", &imageweb.Router{
		Middleware: middleware,
		Hasher:     hasher,
		BlockStore: imageStore,
		Limit:      cfg.Images.Limit,
	})

	srv.AddRouter("object", &objectweb.Router{
		Middleware: middleware,
		Hasher:     hasher,
		BlockStore: objectStore,
		Limit:      cfg.Objects.Limit,
	})

	srv.AddRouter("variable", &variableweb.Router{
		Middleware: middleware,
	})

	srv.AddRouter("key", &keyweb.Router{
		Block:      blockCipher,
		Middleware: middleware,
	})

	srv.AddRouter("oauth2", &oauth2web.Router{
		Block:      blockCipher,
		Middleware: middleware,
	})

	srv.Init(handler)

	if serveui {
		ui := server.UI{
			Server: srv,
			CSRF: csrf.Protect(
				authKey,
				csrf.RequestHeader("X-CSRF-Token"),
				csrf.FieldName("csrf_token"),
			),
		}

		ui.Init()
		ui.Register("auth")
		ui.Register("build", buildweb.Gate(db))
		ui.Register("provider", providerweb.Gate(db))
		ui.Register("namespace", namespaceweb.Gate(db))
		ui.Register("image", imageweb.Gate(db))
		ui.Register("object", objectweb.Gate(db))
		ui.Register("variable", variableweb.Gate(db))
		ui.Register("key", keyweb.Gate(db))
		ui.Register("oauth2")
	}

	var apiPrefix string

	if serveapi {
		if serveui {
			apiPrefix = "/api"
		}

		api := server.API{
			Server: srv,
			Prefix: apiPrefix,
		}

		api.Init()
		api.Register("build", buildweb.Gate(db))
		api.Register("namespace", namespaceweb.Gate(db))
		api.Register("image", imageweb.Gate(db))
		api.Register("object", objectweb.Gate(db))
		api.Register("variable", variableweb.Gate(db))
		api.Register("key", keyweb.Gate(db))
	}

	go func() {
		if err := srv.Serve(); err != nil {
			if cause := errors.Cause(err); cause != http.ErrServerClosed {
				log.Error.Fatal(cause)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*15))
	defer cancel()

	log.Info.Println("djinn-server started on", cfg.Net.Listen)

	if apiPrefix != "" {
		log.Info.Println("api routes being served under", cfg.Net.Listen+apiPrefix)
	}

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	sig := <-c

	srv.Shutdown(ctx)

	if cfg.Pidfile != "" {
		if err := os.RemoveAll(cfg.Pidfile); err != nil {
			log.Error.Println("failed to remove pidfile", err)
		}
	}

	log.Info.Println("signal:", sig, "received, shutting down")
}
