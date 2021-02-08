package config

import (
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"os"

	"github.com/andrewpillar/djinn/fs"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/log"
	"github.com/andrewpillar/djinn/provider"
	"github.com/andrewpillar/djinn/server"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/mcmathja/curlyq"

	"github.com/pelletier/go-toml"

	"github.com/rbcervilla/redisstore"
)

type serverLogger struct {
	log *log.Logger
}

// serverCfg is the representation of a TOML configuration file.
type serverCfg struct {
	Host    string
	Pidfile string

	Net struct {
		Listen string

		SSL struct {
			Cert string
			Key  string
		}
	}

	Crypto cryptoCfg

	SMTP smtpCfg

	Database databaseCfg
	Redis    redisCfg

	Images    storageCfg
	Artifacts storageCfg
	Objects   storageCfg

	Log logCfg

	Drivers []driverCfg

	Providers []providerCfg
}

type Store struct {
	Limit int64
	Store fs.Store
}

// Server contains the necessary resources that the server needs for running.
// These are unmarshalled from a TOML configuration file.
type Server struct {
	pidfile *os.File

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

	log *log.Logger

	producers map[string]*curlyq.Producer

	providers *provider.Registry

	Crypto Crypto
}

var _ curlyq.Logger = (*serverLogger)(nil)

func decodeServer(r io.Reader) (serverCfg, error) {
	var cfg serverCfg

	if err := toml.NewDecoder(r).Decode(&cfg); err != nil {
		return cfg, errors.Err(err)
	}

	hostname, err := os.Hostname()

	if err != nil {
		hostname = "localhost"
	}

	if cfg.Host == "" {
		cfg.Host = hostname
	}

	if cfg.Net.Listen == "" {
		cfg.Net.Listen = "8080"

		if cfg.Net.SSL.Cert != "" && cfg.Net.SSL.Key != "" {
			cfg.Net.Listen = "8443"
		}
	}

	if cfg.Images.Type == "" {
		cfg.Images.Type = "file"
	}

	if cfg.Artifacts.Type == "" {
		cfg.Artifacts.Type = "file"
	}

	if cfg.Objects.Type == "" {
		cfg.Objects.Type = "file"
	}

	if cfg.Log.Level == "" {
		cfg.Log.Level = "INFO"
	}

	if cfg.Log.File == "" {
		cfg.Log.File = "/dev/stdout"
	}
	return cfg, errors.Err(cfg.validate())
}

func DecodeServer(r io.Reader) (Server, error) {
	var s Server

	cfg, err := decodeServer(r)

	if err != nil {
		return s, errors.Err(err)
	}

	s.pidfile, err = mkpidfile(cfg.Pidfile)

	if err != nil {
		return s, errors.Err(err)
	}

	s.Crypto = Crypto{
		Hash:  []byte(cfg.Crypto.Hash),
		Block: []byte(cfg.Crypto.Block),
		Salt:  []byte(cfg.Crypto.Salt),
		Auth:  []byte(cfg.Crypto.Auth),
	}

	s.log = log.New(os.Stdout)
	s.log.SetLevel(cfg.Log.Level)

	if cfg.Log.File != "/dev/stdout" {
		f, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

		if err != nil {
			return s, errors.Err(err)
		}
		s.log.SetWriter(f)
	}

	s.log.Info.Println("logging initiliazed, writing to", cfg.Log.File)

	s.block, err = crypto.NewBlock([]byte(cfg.Crypto.Block), []byte(cfg.Crypto.Salt))

	if err != nil {
		return s, errors.Err(err)
	}

	s.hasher = &crypto.Hasher{
		Salt:   cfg.Crypto.Salt,
		Length: 8,
	}

	if err := s.hasher.Init(); err != nil {
		return s, errors.Err(err)
	}

	s.db, err = connectdb(s.log, cfg.Database)

	if err != nil {
		return s, errors.Err(err)
	}

	s.redis, err = connectredis(s.log, cfg.Redis)

	if err != nil {
		return s, errors.Err(err)
	}

	s.producers = make(map[string]*curlyq.Producer)

	for _, d := range cfg.Drivers {
		s.producers[d.Type] = curlyq.NewProducer(&curlyq.ProducerOpts{
			Client: s.redis,
			Queue:  d.Queue,
			Logger: serverLogger{log: s.log},
		})
	}

	s.smtp, err = connectsmtp(s.log, cfg.SMTP)

	if err != nil {
		return s, errors.Err(err)
	}

	s.postmaster = cfg.SMTP.Admin

	s.images = Store{
		Limit: cfg.Images.Limit,
		Store: blockstores[cfg.Images.Type](cfg.Images.Path, cfg.Images.Limit),
	}

	s.artifacts = Store{
		Limit: cfg.Artifacts.Limit,
		Store: blockstores[cfg.Artifacts.Type](cfg.Artifacts.Path, cfg.Artifacts.Limit),
	}

	s.objects = Store{
		Limit: cfg.Objects.Limit,
		Store: blockstores[cfg.Objects.Type](cfg.Objects.Path, cfg.Objects.Limit),
	}

	if s.images.Store.Init(); err != nil {
		return s, errors.Err(err)
	}

	if s.artifacts.Store.Init(); err != nil {
		return s, errors.Err(err)
	}

	if s.objects.Store.Init(); err != nil {
		return s, errors.Err(err)
	}

	s.providers = provider.NewRegistry()

	for _, p := range cfg.Providers {
		fn, ok := providerFactories[p.Name]

		if !ok {
			return s, errors.New("unknown provider: " + p.Name)
		}
		s.providers.Register(p.Name, fn(cfg.Host, p.Endpoint, p.Secret, p.ClientID, p.ClientSecret))
	}

	redisstore, err := redisstore.NewRedisStore(s.redis)

	if err != nil {
		return s, errors.Err(err)
	}

	url, err := url.Parse(cfg.Host)

	if err != nil {
		return s, errors.Err(err)
	}

	redisstore.KeyPrefix("session_")
	redisstore.KeyGen(func() (string, error) { return cfg.Crypto.Block, nil })
	redisstore.Options(sessions.Options{
		Path:   "/",
		Domain: url.Hostname(),
		MaxAge: 86400 * 60,
	})

	s.session = redisstore

	s.srv = &server.Server{
		Server: &http.Server{
			Addr: cfg.Net.Listen,
		},
		Log:    s.log,
		Router: mux.NewRouter(),
		Cert:   cfg.Net.SSL.Cert,
		Key:    cfg.Net.SSL.Key,
	}
	return s, nil
}

func (cfg serverCfg) validate() error {
	if err := cfg.Crypto.validate(); err != nil {
		return err
	}

	if err := cfg.Database.validate(); err != nil {
		return err
	}

	if cfg.Redis.Addr == "" {
		return errors.New("missing redis address")
	}

	if err := cfg.SMTP.validate(); err != nil {
		return err
	}

	if cfg.Images.Path == "" {
		return errors.New("missing images storage path")
	}

	if cfg.Artifacts.Path == "" {
		return errors.New("missing artifacts storage path")
	}

	if cfg.Objects.Path == "" {
		return errors.New("missing objects storage path")
	}

	if len(cfg.Drivers) == 0 {
		return errors.New("no build drivers configured")
	}
	return nil
}

func (l serverLogger) Debug(v ...interface{}) { l.log.Debug.Println(v...) }
func (l serverLogger) Info(v ...interface{})  { l.log.Info.Println(v...) }
func (l serverLogger) Warn(v ...interface{})  { l.log.Warn.Println(v...) }
func (l serverLogger) Error(v ...interface{}) { l.log.Error.Println(v...) }

func (s Server) Pidfile() *os.File { return s.pidfile }

func (s Server) Server() *server.Server { return s.srv }

func (s Server) DB() *sqlx.DB { return s.db }

func (s Server) Redis() *redis.Client { return s.redis }

func (s Server) SMTP() (*smtp.Client, string) { return s.smtp, s.postmaster }

func (s Server) SessionStore() sessions.Store { return s.session }

func (s Server) Images() Store { return s.images }

func (s Server) Artifacts() Store { return s.artifacts }

func (s Server) Objects() Store { return s.objects }

func (s Server) BlockCipher() *crypto.Block { return s.block }

func (s Server) Hasher() *crypto.Hasher { return s.hasher }

func (s Server) Log() *log.Logger { return s.log }

func (s Server) Producers() map[string]*curlyq.Producer { return s.producers }

func (s Server) Providers() *provider.Registry { return s.providers }
