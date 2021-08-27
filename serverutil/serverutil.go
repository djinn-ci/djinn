package serverutil

import (
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"djinn-ci.com/build"
	"djinn-ci.com/config"
	"djinn-ci.com/cron"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/key"
	"djinn-ci.com/namespace"
	"djinn-ci.com/oauth2"
	"djinn-ci.com/object"
	"djinn-ci.com/queue"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	"djinn-ci.com/variable"
	"djinn-ci.com/version"
	"djinn-ci.com/web"

	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
	"github.com/gorilla/securecookie"

	"github.com/jmoiron/sqlx"

	"github.com/mcmathja/curlyq"

	buildrouter "djinn-ci.com/build/router"
	cronrouter "djinn-ci.com/cron/router"
	imagerouter "djinn-ci.com/image/router"
	keyrouter "djinn-ci.com/key/router"
	namespacerouter "djinn-ci.com/namespace/router"
	oauth2router "djinn-ci.com/oauth2/router"
	objectrouter "djinn-ci.com/object/router"
	providerrouter "djinn-ci.com/provider/router"
	userrouter "djinn-ci.com/user/router"
	variablerouter "djinn-ci.com/variable/router"
)

var (
	uirouters = []string{
		"auth",
		"build",
		"cron",
		"image",
		"key",
		"namespace",
		"oauth2",
		"object",
		"provider",
		"variable",
	}

	apirouters = []string{
		"auth",
		"build",
		"cron",
		"image",
		"key",
		"namespace",
		"object",
		"variable",
	}

	DefaultGates = map[string]func(*sqlx.DB) web.Gate{
		"build":     buildrouter.Gate,
		"cron":      cronrouter.Gate,
		"image":     imagerouter.Gate,
		"key":       keyrouter.Gate,
		"namespace": namespacerouter.Gate,
		"oauth2":    oauth2router.Gate,
		"object":    objectrouter.Gate,
		"provider":  providerrouter.Gate,
		"variable":  variablerouter.Gate,
	}
)

func ParseFlags(args []string) (bool, string, bool, bool) {
	var (
		api     bool
		config  string
		ui      bool
		version bool
	)

	fs := flag.NewFlagSet(args[0], flag.ExitOnError)

	fs.BoolVar(&api, "api", false, "serve only the api endpoints")
	fs.StringVar(&config, "config", "djinn-server.conf", "the config file to use")
	fs.BoolVar(&ui, "ui", false, "serve only the ui endpoints")
	fs.BoolVar(&version, "version", false, "show the version and exit")
	fs.Parse(args[1:])

	return api, config, ui, version
}

func Init(ctx context.Context, path string) (*server.Server, *config.Server, func(), error) {
	env.Load()

	var cfg *config.Server

	f, err := os.Open(path)

	if err != nil {
		return nil, cfg, nil, errors.Err(err)
	}

	defer f.Close()

	cfg, err = config.DecodeServer(f.Name(), f)

	if err != nil {
		return nil, cfg, nil, errors.Err(err)
	}

	pidfile := cfg.Pidfile()

	db := cfg.DB()
	redis := cfg.Redis()
	smtp, postmaster := cfg.SMTP()

	log := cfg.Log()

	close_ := func() {
		db.Close()
		redis.Close()
		smtp.Close()
		log.Close()

		if pidfile != nil {
			if err := os.RemoveAll(pidfile.Name()); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
				os.Exit(1)
			}
		}
	}

	for driver := range cfg.Producers() {
		log.Info.Println("enabled build driver", driver)
	}

	srv := cfg.Server()

	srv.Router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		contentType := r.Header.Get("Content-Type")

		if strings.HasPrefix(accept, "application/json") || strings.HasPrefix(contentType, "application/json") {
			web.JSONError(w, "Not found", http.StatusNotFound)
			return
		}
		web.HTMLError(w, "Not found", http.StatusNotFound)
	})

	srv.Router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		contentType := r.Header.Get("Content-Type")

		if strings.HasPrefix(accept, "application/json") || strings.HasPrefix(accept, contentType) {
			web.JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		web.HTMLError(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	srv.Router.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			webutil.JSON(w, map[string]string{"build": version.Build}, http.StatusOK)
			return
		}
		webutil.Text(w, version.Build, http.StatusOK)
	}).Methods("GET")

	memq := queue.NewMemory(20, func(j queue.Job, err error) {
		log.Error.Println("queue job failed:", j.Name(), err)
	})

	webhooks := namespace.NewWebhookStore(db)

	memq.InitFunc("event:build.submitted", build.InitEvent(webhooks))
	memq.InitFunc("event:build.started", build.InitEvent(webhooks))
	memq.InitFunc("event:build.finished", build.InitEvent(webhooks))
	memq.InitFunc("event:invite.accepted", namespace.InitInviteEvent(webhooks))
	memq.InitFunc("event:invite.sent", namespace.InitInviteEvent(webhooks))
	memq.InitFunc("event:cron", cron.InitEvent(webhooks))
	memq.InitFunc("event:images", image.InitEvent(webhooks))
	memq.InitFunc("event:objects", object.InitEvent(webhooks))
	memq.InitFunc("event:variables", variable.InitEvent(webhooks))
	memq.InitFunc("event:ssh_keys", key.InitEvent(webhooks))

	prd := curlyq.NewProducer(&curlyq.ProducerOpts{
		Client: redis,
		Queue:  "image_downloads",
	})

	gob.Register(&image.DownloadJob{})

	queues := queue.NewSet()
	queues.Add("events", memq)
	queues.Add("image_downloads", queue.NewCurlyQ(log, prd, nil))

	h := web.Handler{
		DB:           db,
		Log:          log,
		Queues:       queues,
		Store:        cfg.SessionStore(),
		SecureCookie: securecookie.New(cfg.Crypto.Hash, cfg.Crypto.Block),
		Users:        user.NewStore(db),
		Tokens:       oauth2.NewTokenStore(db),
	}

	go memq.Consume(ctx)

	h.SMTP.Client = smtp
	h.SMTP.From = postmaster

	mw := web.Middleware{
		Handler: h,
	}

	srv.AddRouter("auth", userrouter.New(cfg, h, mw))
	srv.AddRouter("build", buildrouter.New(cfg, h, mw))
	srv.AddRouter("cron", cronrouter.New(cfg, h, mw))
	srv.AddRouter("image", imagerouter.New(cfg, h, mw))
	srv.AddRouter("key", keyrouter.New(cfg, h, mw))
	srv.AddRouter("namespace", namespacerouter.New(cfg, h, mw))
	srv.AddRouter("oauth2", oauth2router.New(cfg, h, mw))
	srv.AddRouter("object", objectrouter.New(cfg, h, mw))
	srv.AddRouter("provider", providerrouter.New(cfg, h, mw))
	srv.AddRouter("variable", variablerouter.New(cfg, h, mw))
	srv.Init(h.SaveMiddleware)

	return srv, cfg, close_, nil
}

func RegisterRoutes(cfg *config.Server, api, ui bool, srv *server.Server) {
	gates := make(map[string][]web.Gate)

	for name, fn := range DefaultGates {
		gates[name] = []web.Gate{
			fn(cfg.DB()),
		}
	}
	RegisterRoutesWithGates(cfg, api, ui, srv, gates)
}

func RegisterRoutesWithGates(cfg *config.Server, api, ui bool, srv *server.Server, gates map[string][]web.Gate) {
	if !api && !ui {
		api = true
		ui = true
	}

	if ui {
		uisrv := server.UI{
			Server: srv,
			CSRF: csrf.Protect(
				cfg.Crypto.Auth,
				csrf.RequestHeader("X-CSRF-Token"),
				csrf.FieldName("csrf_token"),
			),
		}

		uisrv.Init()

		for _, router := range uirouters {
			uisrv.Register(router, gates[router]...)
		}
	}

	var prefix string

	if api {
		route := "/"

		if ui {
			prefix = "/api"
			route = prefix

			srv.Log.Info.Println("api routes served under", srv.Server.Addr+prefix)
		}

		srv.Router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			addr := webutil.BaseAddress(r)

			data := map[string]string{
				"builds_url":     addr + "/builds",
				"namespaces_url": addr + "/namespaces",
				"cron_url":       addr + "/cron",
				"invites_url":    addr + "/invites",
				"images_url":     addr + "/images",
				"objects_url":    addr + "/objects",
				"variables_url":  addr + "/variables",
				"keys_url":       addr + "/keys",
			}
			webutil.JSON(w, data, http.StatusOK)
		})

		apisrv := server.API{
			Server: srv,
			Prefix: prefix,
		}

		apisrv.Init()

		for _, router := range apirouters {
			apisrv.Register(router, gates[router]...)
		}
	}
}

// Start will start the server in a goroutine. If the server fails to start
// then SIGKILL is sent to the given channel to signal that the program should
// terminate. A channel is used so we can gracefully close down any connections
// that the server may have opened.
func Start(srv *server.Server, ch chan os.Signal) {
	go func() {
		if err := srv.Serve(); err != nil {
			if cause := errors.Cause(err); cause != http.ErrServerClosed {
				srv.Log.Error.Println(cause)

				if nerr, ok := cause.(net.Error); ok && !nerr.Temporary() {
					ch <- os.Kill
				}
			}
		}
	}()
	srv.Log.Info.Println(os.Args[0], "started on", srv.Server.Addr)
}
