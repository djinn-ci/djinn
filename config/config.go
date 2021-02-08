package config

import (
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"

	"github.com/andrewpillar/djinn/fs"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/log"
	"github.com/andrewpillar/djinn/mail"
	"github.com/andrewpillar/djinn/provider"
	"github.com/andrewpillar/djinn/provider/github"
	"github.com/andrewpillar/djinn/provider/gitlab"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"
)

type driverCfg struct {
	Type  string
	Queue string
}

type smtpCfg struct {
	Addr     string
	CA       string
	Admin    string
	Username string
	Password string
}

type cryptoCfg struct {
	Hash  string
	Block string
	Salt  string
	Auth  string
}

type databaseCfg struct {
	Addr     string
	Name     string
	Username string
	Password string
}

type redisCfg struct {
	Addr     string
	Password string
}

type logCfg struct {
	Level string
	File  string
}

type storageCfg struct {
	Type  string
	Path  string
	Limit int64
}

type providerCfg struct {
	Name         string
	Secret       string
	Endpoint     string
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

type Crypto struct {
	Hash  []byte
	Block []byte
	Salt  []byte
	Auth  []byte
}

var (
	blockstores = map[string]func(string, int64) fs.Store{
		"file": func(dsn string, limit int64) fs.Store {
			return fs.NewFilesystemWithLimit(dsn, limit)
		},
	}

	providerFactories = map[string]provider.Factory{
		"github": func(host, endpoint, secret, clientId, clientSecret string) provider.Client {
			return github.New(host, endpoint, secret, clientId, clientSecret)
		},
		"gitlab": func(host, endpoint, secret, clientId, clientSecret string) provider.Client {
			return gitlab.New(host, endpoint, secret, clientId, clientSecret)
		},
	}
)

func connectdb(log *log.Logger, cfg databaseCfg) (*sqlx.DB, error) {
	host, port, err := net.SplitHostPort(cfg.Addr)

	if err != nil {
		return nil, errors.Err(err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		host,
		port,
		cfg.Name,
		cfg.Username,
		cfg.Password,
	)

	log.Debug.Println("connecting to postgresql database with:", dsn)

	db, err := database.Connect(dsn)

	if err != nil {
		return nil, errors.Err(err)
	}

	log.Info.Println("connected to postgresql database")
	return db, nil
}

func connectredis(log *log.Logger, cfg redisCfg) (*redis.Client, error) {
	redis := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
	})

	log.Debug.Println("connecting to redis database with:", cfg.Addr, cfg.Password)

	if _, err := redis.Ping().Result(); err != nil {
		return nil, errors.Err(err)
	}

	log.Info.Println("connected to redis database")
	return redis, nil
}

func connectsmtp(log *log.Logger, cfg smtpCfg) (*smtp.Client, error) {
	log.Debug.Println("connecting to smtp addr", cfg.Addr)

	if cfg.Username != "" && cfg.Password != "" {
		log.Debug.Println("using plain auth username =", cfg.Username, "password =", cfg.Password)
	}

	if cfg.CA != "" {
		log.Debug.Println("connecting to smtp with tls")
	}

	smtp, err := mail.NewClient(mail.ClientConfig{
		CA:       cfg.CA,
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
	})

	if err != nil {
		return nil, errors.Err(err)
	}

	log.Info.Println("connected to smtp server")
	return smtp, nil
}

func mkpidfile(path string) (*os.File, error) {
	if path == "" {
		return nil, nil
	}

	pidfile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0660)

	if err != nil {
		return nil, errors.Err(err)
	}

	pidfile.Write([]byte(strconv.FormatInt(int64(os.Getpid()), 10)))
	pidfile.Close()

	return pidfile, nil
}

func (cfg databaseCfg) validate() error {
	if cfg.Addr == "" {
		return errors.New("missing database address")
	}

	if cfg.Name == "" {
		return errors.New("missing database name")
	}

	if cfg.Username == "" {
		return errors.New("missing database username")
	}

	if cfg.Password == "" {
		return errors.New("missing database password")
	}
	return nil
}

func (cfg smtpCfg) validate() error {
	if cfg.Addr == "" {
		return errors.New("missing smtp address")
	}

	if cfg.Admin == "" {
		return errors.New("missing smtp admin email")
	}
	return nil
}

func (cfg cryptoCfg) validate() error {
	if len(cfg.Hash) < 32 || len(cfg.Hash) > 64 {
		return errors.New("invalid hash key length, must be between 32 and 64 bytes")
	}

	if len(cfg.Block) != 16 && len(cfg.Block) != 24 && len(cfg.Block) != 32 {
		return errors.New("invalid block key, must be either 16, 24, or 32 bytes in length")
	}

	if len(cfg.Auth) != 32 {
		return errors.New("invalid auth key, mustb e 32 bytes")
	}
	return nil
}
