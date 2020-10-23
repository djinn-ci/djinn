package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/djinn/block"
	cronweb "github.com/andrewpillar/djinn/cron/web"
	buildweb "github.com/andrewpillar/djinn/build/web"
	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	imageweb "github.com/andrewpillar/djinn/image/web"
	keyweb "github.com/andrewpillar/djinn/key/web"
	"github.com/andrewpillar/djinn/log"
	"github.com/andrewpillar/djinn/mail"
	namespaceweb "github.com/andrewpillar/djinn/namespace/web"
	"github.com/andrewpillar/djinn/oauth2"
	oauth2web "github.com/andrewpillar/djinn/oauth2/web"
	objectweb "github.com/andrewpillar/djinn/object/web"
	"github.com/andrewpillar/djinn/provider"
	"github.com/andrewpillar/djinn/provider/github"
	"github.com/andrewpillar/djinn/provider/gitlab"
	providerweb "github.com/andrewpillar/djinn/provider/web"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	userweb "github.com/andrewpillar/djinn/user/web"
	variableweb "github.com/andrewpillar/djinn/variable/web"
	"github.com/andrewpillar/djinn/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"

	"github.com/go-redis/redis"

	"github.com/rbcervilla/redisstore"

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

func run(stdout, stderr io.Writer, args []string) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	var (
		showversion bool
		serveui     bool
		serveapi    bool
		configfile  string
	)

	flags.BoolVar(&showversion, "version", false, "show the version and exit")
	flags.BoolVar(&serveui, "ui", false, "serve only the ui endpoints")
	flags.BoolVar(&serveapi, "api", false, "serve only the api endpoints")
	flags.StringVar(&configfile, "config", "djinn-server.toml", "the config file to use")
	flags.Parse(args[1:])

	if showversion {
		fmt.Fprintf(stdout, "%s %s %s\n", args[0], Version, Build)
		return nil
	}

	if !serveui && !serveapi {
		serveui = true
		serveapi = true
	}

	log := log.New(stdout)

	f, err := os.Open(configfile)

	if err != nil {
		return err
	}

	defer f.Close()

	cfg, err := config.DecodeServer(f)

	if err != nil {
		return err
	}

	if len(cfg.Drivers) == 0 {
		return errors.New("no drivers configured")
	}

	if cfg.Pidfile != "" {
		pidf, err := os.OpenFile(cfg.Pidfile, os.O_WRONLY|os.O_CREATE, 0660)

		if err != nil {
			return err
		}

		pidf.Write([]byte(strconv.FormatInt(int64(os.Getpid()), 10)))
		pidf.Close()
	}

	log.SetLevel(cfg.Log.Level)

	logf, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		return err
	}

	defer logf.Close()

	log.Info.Println("logging initialized, writing to", logf.Name())

	log.SetWriter(logf)

	blockCipher, err := crypto.NewBlock([]byte(cfg.Crypto.Block))

	if err != nil {
		return err
	}

	hasher := &crypto.Hasher{
		Salt:   cfg.Crypto.Salt,
		Length: 8,
	}

	if err := hasher.Init(); err != nil {
		return err
	}

	host, port, err := net.SplitHostPort(cfg.Database.Addr)

	if err != nil {
		return err
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
		return err
	}

	log.Info.Println("connected to postgresql database")

	redis := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})

	log.Debug.Println("connecting to redis database with:", cfg.Redis.Addr, cfg.Redis.Password)

	if _, err := redis.Ping().Result(); err != nil {
		return err
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
			return err
		}
		queues[d.Type] = queue
	}

	log.Debug.Println("connecting to smtp addr", cfg.SMTP.Addr)

	if cfg.SMTP.Username != "" && cfg.SMTP.Password != "" {
		log.Debug.Println("connecting to smtp with plain auth username =", cfg.SMTP.Username, "password =", cfg.SMTP.Password)
	}

	if cfg.SMTP.CA != "" {
		log.Debug.Println("connecting to smtp with tls")
	}

	smtp, err := mail.NewClient(mail.ClientConfig{
		CA:       cfg.SMTP.CA,
		Addr:     cfg.SMTP.Addr,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
	})

	if err != nil {
		return err
	}

	log.Info.Println("connected to smtp server")

	hashKey := []byte(cfg.Crypto.Hash)
	blockKey := []byte(cfg.Crypto.Block)

	if len(hashKey) < 32 || len(hashKey) > 64 {
		return errors.New("invalid hash key length, must be between 32 and 64 bytes")
	}

	if len(blockKey) != 16 && len(blockKey) != 24 && len(blockKey) != 32 {
		return errors.New("invalid block key, must be either 16, 24, or 32 bytes")
	}

	imageStore := blockstores[cfg.Images.Type](cfg.Images.Path, cfg.Images.Limit)
	objectStore := blockstores[cfg.Objects.Type](cfg.Objects.Path, cfg.Objects.Limit)
	artifactStore := blockstores[cfg.Artifacts.Type](cfg.Artifacts.Path, cfg.Artifacts.Limit)

	if err := imageStore.Init(); err != nil {
		return err
	}

	if err := objectStore.Init(); err != nil {
		return err
	}

	if err := artifactStore.Init(); err != nil {
		return err
	}

	authKey := []byte(cfg.Crypto.Auth)

	if len(authKey) != 32 {
		return errors.New("invalid auth key, must be 32 bytes")
	}

	providers := provider.NewRegistry()

	for _, p := range cfg.Providers {
		factory, ok := providerFactories[p.Name]

		if !ok {
			return errors.New("unknown provider: " + p.Name)
		}
		providers.Register(
			p.Name,
			factory(cfg.Host, p.Endpoint, p.Secret, p.ClientID, p.ClientSecret),
		)
	}

	store, err := redisstore.NewRedisStore(redis)

	if err != nil {
		return err
	}

	store.KeyPrefix("session_")
	store.KeyGen(func() (string, error) {
		return string(blockKey), nil
	})
	store.Options(sessions.Options{
		Path:  "/",
		Domain: cfg.Host,
		MaxAge: 86400 * 60,
	})

	handler := web.Handler{
		DB:           db,
		Log:          log,
		Store:        store,
		SecureCookie: securecookie.New(hashKey, blockKey),
		Users:        user.NewStore(db),
	}

	handler.SMTP.Client = smtp
	handler.SMTP.From = cfg.SMTP.Admin

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

	srv.AddRouter("cron", &cronweb.Router{
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
		ui.Register("oauth2")
		ui.Register("build", buildweb.Gate(db))
		ui.Register("provider", providerweb.Gate(db))
		ui.Register("namespace", namespaceweb.Gate(db))
		ui.Register("cron", cronweb.Gate(db))
		ui.Register("image", imageweb.Gate(db))
		ui.Register("object", objectweb.Gate(db))
		ui.Register("variable", variableweb.Gate(db))
		ui.Register("key", keyweb.Gate(db))
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
		api.Register("auth")
		api.Register("build", buildweb.Gate(db))
		api.Register("namespace", namespaceweb.Gate(db))
		api.Register("cron", cronweb.Gate(db))
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
			return err
		}
	}

	log.Info.Println("signal:", sig, "received, shutting down")
	return nil
}

func main() {
	if err := run(os.Stdout, os.Stderr, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
