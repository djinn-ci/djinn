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

type sslCfg struct {
	CA   string
	Cert string
	Key  string
}

type databaseCfg struct {
	SSL sslCfg

	Addr     string
	Name     string
	Username string
	Password string
}

type Crypto struct {
	Hash  []byte
	Block []byte
	Salt  []byte
	Auth  []byte
}

type logCfg struct {
	Level string
	File  string
}

type redisCfg struct {
	Addr     string
	Password string
}

type smtpCfg struct {
	Addr     string
	CA       string
	Admin    string
	Username string
	Password string
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

	sslmode := "disable"
	sslconf := ""
	sslcert := cfg.SSL.Cert
	sslkey := cfg.SSL.Key
	sslrootcert := cfg.SSL.CA

	if sslcert != "" && sslkey != "" && sslrootcert != "" {
		sslmode = "verify-full"
		sslconf = fmt.Sprintf("sslcert=%s sslkey=%s sslrootcert=%s", sslcert, sslkey, sslrootcert)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s %s",
		host,
		port,
		cfg.Name,
		cfg.Username,
		cfg.Password,
		sslmode,
		sslconf,
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

func (cfg *providerCfg) put(n *node) error {
	if n.body == nil {
		return n.err("provider must be a configuration block")
	}
	if n.label == "" {
		return n.err("unlabeled provider")
	}

	var walkerr error

	n.body.walk(func(n *node) {
		switch n.name {
		case "secret":
			cfg.Secret = n.value
		case "endpoint":
			cfg.Endpoint = n.value
		case "client_id":
			cfg.ClientID = n.value
		case "client_secret":
			cfg.ClientSecret = n.value
		default:
			walkerr = n.err("unknown provider configuration parameter: " + n.name)
		}
	})
	return walkerr
}

type storeCfg struct {
	Type  string
	Path  string
	Limit int64
}

func (cfg *Crypto) put(n *node) error {
	if n.body == nil {
		return n.err("crypto must be a configuration block")
	}

	var walkerr error

	n.body.walk(func(n *node) {
		if n.body != nil {
			walkerr = n.err("unexpected configuration block")
			return
		}
		if n.list != nil {
			walkerr = n.err("unexpected array")
			return
		}

		switch n.name {
		case "hash":
			cfg.Hash = []byte(n.value)
		case "block":
			cfg.Block = []byte(n.value)
		case "salt":
			cfg.Salt = []byte(n.value)
		case "auth":
			cfg.Auth = []byte(n.value)
		default:
			walkerr = n.err("unknown crypto configuration parameter: " + n.name)
		}
	})
	return walkerr
}

func (cfg *logCfg) put(n *node) error {
	levels := map[string]struct{}{
		"debug": {},
		"info":  {},
		"warn":  {},
		"error": {},
	}

	if n.label == "" {
		return n.err("log level not set")
	}

	if _, ok := levels[n.label]; !ok {
		return n.err("unknown log level: " + n.label)
	}

	cfg.Level = n.label
	cfg.File = n.value
	return nil
}

func (cfg *sslCfg) put(n *node) error {
	if n.body == nil {
		return n.err("ssl must be a configuration block")
	}

	var walkerr error

	n.body.walk(func(n *node) {
		if n.body != nil {
			walkerr = n.err("unexpected configuration block")
			return
		}
		if n.list != nil {
			walkerr = n.err("unexpected array")
			return
		}

		switch n.name {
		case "ca":
			cfg.CA = n.value
		case "cert":
			cfg.Cert = n.value
		case "key":
			cfg.Key = n.value
		default:
			walkerr = n.err("unknown ssl configuration parameter: " + n.name)
		}
	})
	return walkerr
}

func (cfg *databaseCfg) put(n *node) error {
	if n.body == nil {
		return n.err("database must be a configuration block")
	}

	var walkerr error

	n.body.walk(func(n *node) {
		if n.body != nil {
			walkerr = n.err("unexpected configuration block")
			return
		}
		if n.list != nil {
			walkerr = n.err("unexpected array")
			return
		}

		switch n.name {
		case "addr":
			cfg.Addr = n.value
		case "name":
			cfg.Name = n.value
		case "username":
			cfg.Username = n.value
		case "password":
			cfg.Password = n.value
		case "ssl":
			walkerr = cfg.SSL.put(n)
		case "ca", "cert", "key":
			return
		default:
			walkerr = n.err("unknown database configuration parameter: " + n.name)
		}
	})
	return walkerr
}

func (cfg *smtpCfg) put(n *node) error {
	if n.body == nil {
		return n.err("smtp must be a configuration block")
	}

	var walkerr error

	n.body.walk(func(n *node) {
		if n.body != nil {
			walkerr = n.err("unexpected configuration block")
			return
		}
		if n.list != nil {
			walkerr = n.err("unexpected array")
			return
		}

		switch n.name {
		case "addr":
			cfg.Addr = n.value
		case "ca":
			cfg.CA = n.value
		case "admin":
			cfg.Admin = n.value
		case "username":
			cfg.Username = n.value
		case "password":
			cfg.Password = n.value
		default:
			walkerr = n.err("unknown smtp configuration parameter: " + n.name)
		}
	})
	return walkerr
}

func (cfg *redisCfg) put(n *node) error {
	if n.body == nil {
		return n.err("redis must be a configuration block")
	}

	var walkerr error

	n.body.walk(func(n *node) {
		if n.body != nil {
			walkerr = n.err("unexpected configuration block")
			return
		}
		if n.list != nil {
			walkerr = n.err("unexpected array")
			return
		}

		switch n.name {
		case "addr":
			cfg.Addr = n.value
		case "password":
			cfg.Password = n.value
		default:
			walkerr = n.err("unknown redis configuration parameter: " + n.name)
		}
	})
	return walkerr
}

func (cfg *storeCfg) put(n *node) error {
	if n.body == nil {
		return n.err("store must be a configuration block")
	}
	if n.label == "" {
		return n.err("unlabeled store")
	}

	var walkerr error

	n.body.walk(func(n *node) {
		if n.body != nil {
			walkerr = n.err("unexpected configuration block")
			return
		}
		if n.list != nil {
			walkerr = n.err("unexpected array")
			return
		}

		switch n.name {
		case "type":
			cfg.Type = n.value
		case "path":
			cfg.Path = n.value
		case "limit":
			if n.lit != numberLit {
				walkerr = n.err("store limit is not a valid integer")
				return
			}

			i, err := strconv.ParseInt(n.value, 10, 64)

			if err != nil {
				walkerr = n.err("store limit is not a valid integer")
				return
			}
			cfg.Limit = i
		default:
			walkerr = n.err("unknown store configuration parameter: " + n.name)
		}
	})
	return walkerr
}
