package serverutil

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/gob"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"

	"djinn-ci.com/build"
	"djinn-ci.com/config"
	"djinn-ci.com/cron"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/key"
	"djinn-ci.com/mail"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"
	"djinn-ci.com/queue"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/webutil/v2"

	"github.com/mcmathja/curlyq"

	buildhttp "djinn-ci.com/build/http"
	cronhttp "djinn-ci.com/cron/http"
	imagehttp "djinn-ci.com/image/http"
	keyhttp "djinn-ci.com/key/http"
	namespacehttp "djinn-ci.com/namespace/http"
	oauth2http "djinn-ci.com/oauth2/http"
	objecthttp "djinn-ci.com/object/http"
	providerhttp "djinn-ci.com/provider/http"
	userhttp "djinn-ci.com/user/http"
	variablehttp "djinn-ci.com/variable/http"
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

func Init(ctx context.Context, path string) (*server.Server, func(), error) {
	env.Load()

	f, err := os.Open(path)

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	defer f.Close()

	cfg, err := config.DecodeServer(f.Name(), f)

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	srv, err := server.New(cfg)

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	if srv.Debug {
		srv.Log.Info.Println("debug mode enabled")
	}

	pidfile := cfg.Pidfile()

	smtp, _ := cfg.SMTP()

	close := func() {
		srv.DB.Close()
		srv.Redis.Close()
		smtp.Close()
		srv.Log.Close()

		if pidfile != "" {
			if err := os.RemoveAll(pidfile); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
				os.Exit(1)
			}
		}
	}

	for driver := range cfg.DriverQueues() {
		srv.Log.Info.Println("enabled build driver", driver)
	}

	memq := queue.NewMemory(20, func(j queue.Job, err error) {
		srv.Log.Error.Println("queue job failed:", j.Name(), err)
	})

	webhooks := &namespace.WebhookStore{
		Store:  namespace.NewWebhookStore(srv.DB),
		AESGCM: srv.AESGCM,
	}

	memq.InitFunc("event:build.submitted", build.InitEvent(webhooks))
	memq.InitFunc("event:build.started", build.InitEvent(webhooks))
	memq.InitFunc("event:build.tagged", build.InitTagEvent(webhooks))
	memq.InitFunc("event:build.finished", build.InitEvent(webhooks))
	memq.InitFunc("event:build.pinned", build.InitEvent(webhooks))
	memq.InitFunc("event:build.unpinned", build.InitEvent(webhooks))
	memq.InitFunc("event:invite.accepted", namespace.InitInviteEvent(srv.DB, webhooks))
	memq.InitFunc("event:invite.rejected", namespace.InitInviteEvent(srv.DB, webhooks))
	memq.InitFunc("event:invite.sent", namespace.InitInviteEvent(srv.DB, webhooks))
	memq.InitFunc("event:namespaces", namespace.InitEvent(webhooks))
	memq.InitFunc("event:cron", cron.InitEvent(webhooks))
	memq.InitFunc("event:images", image.InitEvent(webhooks))
	memq.InitFunc("event:objects", object.InitEvent(webhooks))
	memq.InitFunc("event:variables", variable.InitEvent(webhooks))
	memq.InitFunc("event:ssh_keys", key.InitEvent(webhooks))

	memq.InitFunc("email", mail.InitJob(srv.SMTP.Client))

	gob.Register(&image.DownloadJob{})

	srv.Queues.Add("email", memq)
	srv.Queues.Add("events", memq)
	srv.Queues.Add("jobs", queue.NewRedisProducer(srv.Log, &curlyq.ProducerOpts{
		Client: srv.Redis,
	}))

	go memq.Consume(ctx)
	go namespace.PruneWebhookDeliveries(ctx, srv.Log, srv.DB)

	return srv, close, nil
}

//go:embed favicon.ico
var favicon []byte

func RegisterRoutes(api, ui bool, srv *server.Server) {
	if !api && !ui {
		api = true
		ui = true
	}

	auth, err := srv.Auths.Get(user.InternalProvider)

	if err != nil {
		panic("serverutil: no internal authenticator registered")
	}

	buf := make([]byte, len(favicon))

	if _, err := base64.StdEncoding.Decode(buf, favicon); err != nil {
		panic("serverutil: failed to decode favicon")
	}

	srv.Router.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(buf)
	})

	if ui {
		srv.Log.Debug.Println("registering ui routes")

		userhttp.RegisterUI(auth, srv)
		buildhttp.RegisterUI(auth, srv)
		cronhttp.RegisterUI(auth, srv)
		imagehttp.RegisterUI(auth, srv)
		keyhttp.RegisterUI(auth, srv)
		namespacehttp.RegisterUI(auth, srv)
		oauth2http.RegisterUI(auth, srv)
		objecthttp.RegisterUI(auth, srv)
		providerhttp.RegisterUI(auth, srv)
		providerhttp.RegisterHooks(srv)
		variablehttp.RegisterUI(auth, srv)
	}

	apiPrefix := "/"

	if api {
		if ui {
			apiPrefix = "/api"
			env.DJINN_API_SERVER += apiPrefix

			srv.Log.Info.Println("api routes served under", srv.Server.Addr+apiPrefix)
		}

		srv.Log.Debug.Println("registering api routes")

		router := srv.Router

		srv.Router = srv.Router.PathPrefix(apiPrefix).Subrouter()

		userhttp.RegisterAPI(auth, srv)
		buildhttp.RegisterAPI(auth, srv)
		cronhttp.RegisterAPI(auth, srv)
		imagehttp.RegisterAPI(auth, srv)
		keyhttp.RegisterAPI(auth, srv)
		namespacehttp.RegisterAPI(auth, srv)
		objecthttp.RegisterAPI(auth, srv)
		variablehttp.RegisterAPI(auth, srv)

		srv.Router.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			webutil.JSON(w, map[string]string{
				"builds_url":     env.DJINN_API_SERVER + "/builds",
				"namespaces_url": env.DJINN_API_SERVER + "/namespaces",
				"cron_url":       env.DJINN_API_SERVER + "/cron",
				"invites_url":    env.DJINN_API_SERVER + "/invites",
				"images_url":     env.DJINN_API_SERVER + "/images",
				"objects_url":    env.DJINN_API_SERVER + "/objects",
				"variables_url":  env.DJINN_API_SERVER + "/variables",
				"keys_url":       env.DJINN_API_SERVER + "/keys",
			}, http.StatusOK)
		})

		srv.Router = router
	}

	srv.Router.Use(srv.Save)
	srv.Init()
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
					return
				}
			}
		}
	}()

	srv.Log.Info.Println(os.Args[0], "started on", srv.Server.Addr)
}
