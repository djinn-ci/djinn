package serverutil

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/oauth2"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/securecookie"

	"github.com/jmoiron/sqlx"

	buildrouter "github.com/andrewpillar/djinn/build/router"
	cronrouter "github.com/andrewpillar/djinn/cron/router"
	imagerouter "github.com/andrewpillar/djinn/image/router"
	keyrouter "github.com/andrewpillar/djinn/key/router"
	namespacerouter "github.com/andrewpillar/djinn/namespace/router"
	oauth2router "github.com/andrewpillar/djinn/oauth2/router"
	objectrouter "github.com/andrewpillar/djinn/object/router"
	providerrouter "github.com/andrewpillar/djinn/provider/router"
	userrouter "github.com/andrewpillar/djinn/user/router"
	variablerouter "github.com/andrewpillar/djinn/variable/router"
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
	fs.StringVar(&config, "config", "djinn-server.toml", "the config file to use")
	fs.BoolVar(&ui, "ui", false, "serve only the ui endpoints")
	fs.BoolVar(&version, "version", false, "show the version and exit")
	fs.Parse(args[1:])

	return api, config, ui, version
}

func Init(path string) (*server.Server, config.Server, func(), error) {
	var cfg config.Server

	f, err := os.Open(path)

	if err != nil {
		return nil, cfg, nil, errors.Err(err)
	}

	defer f.Close()

	cfg, err = config.DecodeServer(f)

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

	srv := cfg.Server()

	srv.Router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			web.JSONError(w, "Not found", http.StatusNotFound)
			return
		}
		web.HTMLError(w, "Not found", http.StatusNotFound)
	})

	srv.Router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			web.JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		web.HTMLError(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	h := web.Handler{
		DB:           db,
		Log:          log,
		Store:        cfg.SessionStore(),
		SecureCookie: securecookie.New(cfg.Crypto.Hash, cfg.Crypto.Block),
		Users:        user.NewStore(db),
	}

	h.SMTP.Client = smtp
	h.SMTP.From = postmaster

	mw := web.Middleware{
		Handler: h,
		Tokens:  oauth2.NewTokenStore(db),
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
	srv.Init()

	return srv, cfg, close_, nil
}

func RegisterRoutes(cfg config.Server, api, ui bool, srv *server.Server) {
	gates := make(map[string][]web.Gate)

	for name, fn := range DefaultGates {
		gates[name] = []web.Gate{
			fn(cfg.DB()),
		}
	}
	RegisterRoutesWithGates(cfg, api, ui, srv, gates)
}

func RegisterRoutesWithGates(cfg config.Server, api, ui bool, srv *server.Server, gates map[string][]web.Gate) {
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
		if ui {
			prefix = "/api"

			srv.Log.Info.Println("api routes served under", srv.Server.Addr+prefix)
		}

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

func Start(srv *server.Server) {
	go func() {
		if err := srv.Serve(); err != nil {
			if cause := errors.Cause(err); cause != http.ErrServerClosed {
				srv.Log.Error.Println(cause)
			}
		}
	}()
	srv.Log.Info.Println(os.Args[0], "started on", srv.Server.Addr)
}
