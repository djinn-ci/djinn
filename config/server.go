package config

import (
	"io"
	"net/http"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/log"
	"djinn-ci.com/mail"
	"djinn-ci.com/provider"

	"github.com/andrewpillar/config"
	"github.com/andrewpillar/fs"

	"github.com/go-redis/redis"

	"github.com/mcmathja/curlyq"
)

type serverCfg struct {
	Debug   bool
	Host    string
	Pidfile string

	Log map[string]string

	Drivers []string

	Net struct {
		Listen string

		WriteTimeout time.Duration `config:"write_timeout"`
		ReadTimeout  time.Duration `config:"read_timeout"`

		TLS tlsCfg
		SSL tlsCfg `config:"ssl,deprecated:tls"`
	}

	Crypto cryptoCfg

	SMTP smtpCfg

	Database databaseCfg
	Redis    redisCfg

	Store map[string]storeCfg

	Provider map[string]providerCfg
}

type Server struct {
	debug   bool
	host    string
	pidfile string

	log *log.Logger

	driverQueues map[string]*curlyq.Producer

	srv *http.Server

	crypto cryptoCfg
	aesgcm *crypto.AESGCM
	hasher *crypto.Hasher

	db    *database.Pool
	redis *redis.Client

	smtp      *mail.Client
	smtpadmin string

	artifacts fs.FS
	images    fs.FS
	objects   fs.FS

	auths     *auth.Registry
	providers *provider.Registry
}

func (s *Server) Debug() bool                               { return s.debug }
func (s *Server) Host() string                              { return s.host }
func (s *Server) Pidfile() string                           { return s.pidfile }
func (s *Server) Log() *log.Logger                          { return s.log }
func (s *Server) Server() *http.Server                      { return s.srv }
func (s *Server) DB() *database.Pool                        { return s.db }
func (s *Server) Redis() *redis.Client                      { return s.redis }
func (s *Server) SMTP() (*mail.Client, string)              { return s.smtp, s.smtpadmin }
func (s *Server) DriverQueues() map[string]*curlyq.Producer { return s.driverQueues }
func (s *Server) AESGCM() *crypto.AESGCM                    { return s.aesgcm }
func (s *Server) Hasher() *crypto.Hasher                    { return s.hasher }
func (s *Server) Crypto() ([]byte, []byte, []byte, []byte)  { return s.crypto.values() }
func (s *Server) Artifacts() fs.FS                          { return s.artifacts }
func (s *Server) Images() fs.FS                             { return s.images }
func (s *Server) Objects() fs.FS                            { return s.objects }
func (s *Server) Auths() *auth.Registry                     { return s.auths }
func (s *Server) Providers() *provider.Registry             { return s.providers }

func DecodeServer(name string, r io.Reader) (*Server, error) {
	var cfg serverCfg

	dec := config.NewDecoder(name, decodeOpts...)

	if err := dec.Decode(&cfg, r); err != nil {
		return nil, err
	}

	srv := &Server{
		debug: cfg.Debug,
		host:  cfg.Host,
	}

	var err error

	srv.pidfile, err = mkpidfile(cfg.Pidfile)

	if err != nil {
		return nil, err
	}

	srv.log, err = logger(cfg.Log)

	srv.crypto = cfg.Crypto

	srv.aesgcm, err = cfg.Crypto.aesgcm()

	if err != nil {
		return nil, err
	}

	srv.hasher, err = cfg.Crypto.hasher()

	if err != nil {
		return nil, err
	}

	srv.srv = &http.Server{
		Addr:         cfg.Net.Listen,
		WriteTimeout: cfg.Net.WriteTimeout,
		ReadTimeout:  cfg.Net.ReadTimeout,
	}

	srv.srv.TLSConfig, err = cfg.Net.TLS.config()

	if err != nil {
		return nil, err
	}

	srv.db, err = cfg.Database.connect(srv.log)

	if err != nil {
		return nil, err
	}

	srv.redis, err = cfg.Redis.connect(srv.log)

	if err != nil {
		return nil, err
	}

	srv.driverQueues = driverQueues(srv.log, srv.redis, cfg.Drivers)

	srv.smtp, srv.smtpadmin, err = cfg.SMTP.connect(srv.log)

	if err != nil {
		return nil, err
	}

	for _, label := range []string{"artifacts", "images", "objects"} {
		s, ok := cfg.Store[label]

		if !ok {
			return nil, errors.New(label + " store not configured")
		}

		var err error

		switch label {
		case "artifacts":
			srv.artifacts, err = s.store()
		case "images":
			srv.images, err = s.store()
		case "objects":
			srv.objects, err = s.store()
		}

		if err != nil {
			return nil, err
		}
	}

	auths := auth.NewRegistry("auth_mech")

	srv.providers = &provider.Registry{}

	for name, p := range cfg.Provider {
		a, err := p.auth(name, cfg.Host)

		if err != nil {
			return nil, err
		}

		auths.Register("oauth2."+name, a)

		cli, err := p.client(name, cfg.Host)

		if err != nil {
			return nil, err
		}
		srv.providers.Register(name, cli)
	}

	srv.auths = auths
	return srv, nil
}
