package config

import (
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"os"

	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/fs"
	"github.com/andrewpillar/djinn/log"
	"github.com/andrewpillar/djinn/provider"
	"github.com/andrewpillar/djinn/server"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/mcmathja/curlyq"

	"github.com/rbcervilla/redisstore"
)

type providerCfg struct {
	Secret       string
	Endpoint     string
	ClientID     string
	ClientSecret string
}

type serverCfg struct {
	Host    string
	Pidfile string

	Log logCfg

	Drivers    []string
	ShareQueue bool

	Net struct {
		Listen string

		SSL sslCfg
	}

	Crypto Crypto

	SMTP smtpCfg

	Database databaseCfg
	Redis    redisCfg

	Stores    map[string]storeCfg
	Providers map[string]providerCfg
}

type serverLogger struct {
	log *log.Logger
}

type Store struct {
	Limit int64
	Store fs.Store
}

type Server struct {
	pidfile *os.File

	log *log.Logger

	srv *server.Server

	block  *crypto.Block
	hasher *crypto.Hasher

	db         *sqlx.DB
	redis      *redis.Client
	smtp       *smtp.Client
	postmaster string
	session    sessions.Store

	images    Store
	artifacts Store
	objects   Store

	producers map[string]*curlyq.Producer

	providers *provider.Registry

	Crypto Crypto
}

var (
	_ curlyq.Logger = (*serverLogger)(nil)

	defaultBuildQueue = "builds"
)

func DecodeServer(name string, r io.Reader) (*Server, error) {
	errh := func(name string, line, col int, msg string) {
		fmt.Fprintf(os.Stderr, "%s,%d:%d - %s\n", name, line, col, msg)
	}

	p := newParser(name, r, errh)

	nodes := p.parse()

	if err := p.err(); err != nil {
		return nil, err
	}

	var cfg0 serverCfg

	for _, n := range nodes {
		if err := cfg0.put(n); err != nil {
			return nil, err
		}
	}

	var err error

	cfg := &Server{}
	cfg.pidfile, err = mkpidfile(cfg0.Pidfile)

	if err != nil {
		return nil, err
	}

	cfg.Crypto = cfg0.Crypto
	cfg.log = log.New(os.Stdout)
	cfg.log.SetLevel(cfg0.Log.Level)

	if cfg0.Log.File != "/dev/stdout" {
		f, err := os.OpenFile(cfg0.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0640)

		if err != nil {
			return nil, err
		}
		cfg.log.SetWriter(f)
	}

	cfg.log.Info.Println("logging initialized, writing to", cfg0.Log.File)

	cfg.block, err = crypto.NewBlock(cfg.Crypto.Block, cfg.Crypto.Salt)

	if err != nil {
		return nil, err
	}

	cfg.hasher = &crypto.Hasher{
		Salt:   string(cfg.Crypto.Salt),
		Length: 8,
	}

	if err := cfg.hasher.Init(); err != nil {
		return nil, err
	}

	cfg.db, err = connectdb(cfg.log, cfg0.Database)

	if err != nil {
		return nil, err
	}

	cfg.redis, err = connectredis(cfg.log, cfg0.Redis)

	if err != nil {
		return nil, err
	}

	cfg.producers = make(map[string]*curlyq.Producer)

	for _, driver := range cfg0.Drivers {
		queue := defaultBuildQueue + "_" + driver

		if cfg0.ShareQueue {
			queue = defaultBuildQueue
		}

		cfg.producers[driver] = curlyq.NewProducer(&curlyq.ProducerOpts{
			Client: cfg.redis,
			Queue:  queue,
			Logger: serverLogger{log: cfg.log},
		})
	}

	cfg.smtp, err = connectsmtp(cfg.log, cfg0.SMTP)

	if err != nil {
		return nil, err
	}

	cfg.postmaster = cfg0.SMTP.Admin

	store, ok := cfg0.Stores["images"]

	if !ok {
		return nil, errors.New("image store not configured")
	}

	cfg.images = Store{
		Limit: store.Limit,
		Store: blockstores[store.Type](store.Path, store.Limit),
	}

	store, ok = cfg0.Stores["artifacts"]

	if !ok {
		return nil, errors.New("artifact store not configured")
	}

	cfg.artifacts = Store{
		Limit: store.Limit,
		Store: blockstores[store.Type](store.Path, store.Limit),
	}

	store, ok = cfg0.Stores["objects"]

	if !ok {
		return nil, errors.New("object store not configured")
	}

	cfg.objects = Store{
		Limit: store.Limit,
		Store: blockstores[store.Type](store.Path, store.Limit),
	}

	if err := cfg.images.Store.Init(); err != nil {
		return nil, err
	}
	if err := cfg.artifacts.Store.Init(); err != nil {
		return nil, err
	}
	if err := cfg.objects.Store.Init(); err != nil {
		return nil, err
	}

	cfg.providers = provider.NewRegistry()

	for name, prv := range cfg0.Providers {
		fn, ok := providerFactories[name]

		if !ok {
			return nil, errors.New("unknown provider: " + name)
		}
		cfg.providers.Register(name, fn(cfg0.Host, prv.Endpoint, prv.Secret, prv.ClientID, prv.ClientSecret))
	}

	redisstore, err := redisstore.NewRedisStore(cfg.redis)

	if err != nil {
		return nil, err
	}

	url, err := url.Parse(cfg0.Host)

	if err != nil {
		return nil, errors.New("invalid host url")
	}

	redisstore.KeyPrefix("session_")
	redisstore.KeyGen(func() (string, error) {
		return string(cfg.Crypto.Block), nil
	})
	redisstore.Options(sessions.Options{
		Path:   "/",
		Domain: url.Hostname(),
		MaxAge: 86400*60,
	})

	cfg.session = redisstore
	cfg.srv = &server.Server{
		Server: &http.Server{
			Addr: cfg0.Net.Listen,
		},
		Log:    cfg.log,
		Router: mux.NewRouter(),
		Cert:   cfg0.Net.SSL.Cert,
		Key:    cfg0.Net.SSL.Key,
	}
	return cfg, nil
}

func (s *serverCfg) put(n *node) error {
	switch n.name {
	case "host":
		if n.lit != stringLit {
			return n.err("host must be a string")
		}
		s.Host = n.value
	case "pidfile":
		if n.lit != stringLit {
			return n.err("pidfile must be a string")
		}
		s.Pidfile = n.value
	case "log":
		return s.Log.put(n)
	case "drivers":
		if n.list == nil {
			return n.err("drivers must be an array")
		}

		var walkerr error

		n.list.walk(func(n *node) {
			if n.lit != stringLit {
				walkerr = n.err("drivers must be an array of strings")
				return
			}
			s.Drivers = append(s.Drivers, n.value)
		})

		if walkerr != nil {
			return walkerr
		}
	case "share_queue":
		if n.lit != boolLit {
			return n.err("share_queue must be a boolean")
		}

		if n.value == "true" {
			s.ShareQueue = true
			return nil
		}
		s.ShareQueue = false
	case "net":
		if n.body == nil {
			return n.err("net must be a configuration block")
		}

		var walkerr error

		n.body.walk(func(n *node) {
			switch n.name {
			case "listen":
				s.Net.Listen = n.value
			case "ssl":
				s.Net.SSL.put(n)
			case "cert", "key":
				return
			default:
				walkerr = n.err("unknown net configuration parameter: " + n.name)
			}
		})
		return walkerr
	case "crypto":
		return s.Crypto.put(n)
	case "smtp":
		return s.SMTP.put(n)
	case "database":
		return s.Database.put(n)
	case "redis":
		return s.Redis.put(n)
	case "store":
		var cfg storeCfg

		if err := cfg.put(n); err != nil {
			return err
		}

		if s.Stores == nil {
			s.Stores = make(map[string]storeCfg)
		}
		s.Stores[n.label] = cfg
	case "provider":
		var cfg providerCfg

		if s.Providers == nil {
			s.Providers = make(map[string]providerCfg)
		}
		if err := cfg.put(n); err != nil {
			return err
		}
		s.Providers[n.label] = cfg
	default:
		return n.err("unknown configuration parameter: " + n.name)
	}
	return nil
}

func (l serverLogger) Debug(v ...interface{}) { l.log.Debug.Println(v...) }
func (l serverLogger) Info(v ...interface{})  { l.log.Info.Println(v...) }
func (l serverLogger) Warn(v ...interface{})  { l.log.Warn.Println(v...) }
func (l serverLogger) Error(v ...interface{}) { l.log.Error.Println(v...) }

func (s *Server) Pidfile() *os.File { return s.pidfile }
func (s *Server) Server() *server.Server { return s.srv }
func (s *Server) DB() *sqlx.DB { return s.db }
func (s *Server) Redis() *redis.Client { return s.redis }
func (s *Server) SMTP() (*smtp.Client, string) { return s.smtp, s.postmaster }
func (s *Server) SessionStore() sessions.Store { return s.session }
func (s *Server) Images() Store { return s.images }
func (s *Server) Artifacts() Store { return s.artifacts }
func (s *Server) Objects() Store { return s.objects }
func (s *Server) BlockCipher() *crypto.Block { return s.block }
func (s *Server) Hasher() *crypto.Hasher { return s.hasher }
func (s *Server) Log() *log.Logger { return s.log }
func (s *Server) Producers() map[string]*curlyq.Producer { return s.producers }
func (s *Server) Providers() *provider.Registry { return s.providers }
