package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"os"
	"strconv"

	"djinn-ci.com/auth"
	"djinn-ci.com/auth/oauth2"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/log"
	"djinn-ci.com/mail"
	"djinn-ci.com/provider"
	"djinn-ci.com/provider/github"
	"djinn-ci.com/provider/gitlab"

	"github.com/andrewpillar/config"
	"github.com/andrewpillar/fs"

	"github.com/go-redis/redis"

	"github.com/mcmathja/curlyq"
)

var (
	decodeOpts = []config.Option{
		config.Envvars,
		config.Includes,
	}

	defaultBuildQueue = "builds"
)

// driverQueues returns a map containing each of the queues that builds would be
// produced to, based on the driver being used for that build.
func driverQueues(l *log.Logger, redis *redis.Client, drivers []string) map[string]*curlyq.Producer {
	queues := make(map[string]*curlyq.Producer)

	for _, driver := range drivers {
		queues[driver] = curlyq.NewProducer(&curlyq.ProducerOpts{
			Client: redis,
			Queue:  defaultBuildQueue + "_" + driver,
			Logger: log.Queue{Logger: l},
		})
	}
	return queues
}

// mkpidfile creates a file at the given path, and writes the current PID to
// that file. If the given path is empty, then nothing happens, otherwise if
// the file is created successfully, then the full path to that file is
// returned.
func mkpidfile(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	pidfile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)

	if err != nil {
		return "", err
	}

	if _, err := io.WriteString(pidfile, strconv.FormatInt(int64(os.Getpid()), 10)); err != nil {
		return "", err
	}

	pidfile.Close()
	return pidfile.Name(), nil
}

type cryptoCfg struct {
	Auth  string
	Block string
	Hash  string
	Salt  string
}

func (cfg *cryptoCfg) aesgcm() (*crypto.AESGCM, error) {
	if l := len(cfg.Block); l != 16 && l != 24 && l != 32 {
		return nil, errors.New("crypto block must be 16, 24, or 32 characters in length")
	}

	if len(cfg.Salt) == 0 {
		return nil, errors.New("crypto salt is required for AES-GCM encryption")
	}

	aesgcm, err := crypto.NewAESGCM([]byte(cfg.Block), []byte(cfg.Salt))

	if err != nil {
		return nil, err
	}
	return aesgcm, nil
}

// values simply returns all of the underlying configuration parameters for
// crypto. These are returned in lexical order, so, auth, block, hash, then
// salt.
func (cfg *cryptoCfg) values() ([]byte, []byte, []byte, []byte) {
	return []byte(cfg.Auth), []byte(cfg.Block), []byte(cfg.Hash), []byte(cfg.Salt)
}

// Hasher configures a new crypto.Hasher using the salt parameter set on the
// given crypto configuration.
func (cfg *cryptoCfg) hasher() (*crypto.Hasher, error) {
	if len(cfg.Salt) == 0 {
		return nil, errors.New("crypto salt is required for hashing")
	}

	hasher := &crypto.Hasher{
		Salt:   cfg.Salt,
		Length: 8,
	}

	if err := hasher.Init(); err != nil {
		return nil, err
	}
	return hasher, nil
}

var logmask = os.O_WRONLY | os.O_APPEND | os.O_CREATE

func logger(logtab map[string]string) (*log.Logger, error) {
	var (
		level log.Level
		file  string
	)

	for label, val := range logtab {
		lvl, ok := log.Levels[label]

		if !ok {
			return nil, errors.New("unknown log level " + label)
		}

		if lvl <= level || level == 0 {
			level = lvl
			file = val
		}
	}

	log := log.New(os.Stdout)
	log.SetLevel(level.String())

	if file != os.Stdout.Name() {
		f, err := os.OpenFile(file, logmask, 0640)

		if err != nil {
			return nil, err
		}
		log.SetWriter(f)
	}

	log.Info.Println("logging initialized, writing to", file, "at level", level.String())

	return log, nil
}

type tlsCfg struct {
	CA   string
	Cert string
	Key  string
}

// Config configurates a new tls.Config based on the given ssl configuration.
func (cfg *tlsCfg) config() (*tls.Config, error) {
	var (
		rootCAs *x509.CertPool
		certs   []tls.Certificate
	)

	if cfg.CA != "" {
		b, err := os.ReadFile(cfg.CA)

		if err != nil {
			return nil, err
		}

		pool := x509.NewCertPool()

		if !pool.AppendCertsFromPEM(b) {
			return nil, errors.New("failed to append certificates from PEM, please check if valid")
		}
		rootCAs = pool
	}

	if cfg.Cert != "" && cfg.Key != "" {
		cert, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)

		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}

	if rootCAs != nil || len(certs) > 0 {
		return &tls.Config{
			RootCAs:      rootCAs,
			Certificates: certs,
		}, nil
	}
	return nil, nil
}

type databaseCfg struct {
	TLS tlsCfg
	SSL tlsCfg `config:"ssl,deprecated:tls"`

	Addr     string
	Name     string
	Username string
	Password string
}

var dsnfmt = "host=%s port=%s dbname=%s user=%s password=%s sslmode=%s %s"

func (cfg *databaseCfg) connect(log *log.Logger) (*database.Pool, error) {
	host, port, err := net.SplitHostPort(cfg.Addr)

	if err != nil {
		return nil, err
	}

	sslmode := "disable"
	sslconf := ""
	sslcert := cfg.SSL.Cert
	sslkey := cfg.SSL.Key
	sslrootcert := cfg.SSL.CA

	if cfg.TLS.Cert != "" {
		sslcert = cfg.TLS.Cert
	}
	if cfg.TLS.Key != "" {
		sslcert = cfg.TLS.Key
	}
	if cfg.TLS.CA != "" {
		sslcert = cfg.TLS.CA
	}

	if sslcert != "" && sslkey != "" && sslrootcert != "" {
		sslmode = "verify-full"
		sslconf = fmt.Sprintf("sslcert=%s sslkey=%s sslrootcert=%s", sslcert, sslkey, sslrootcert)
	}

	dsn := fmt.Sprintf(dsnfmt, host, port, cfg.Name, cfg.Username, cfg.Password, sslmode, sslconf)

	log.Debug.Println("connecting to postgresql database with:", dsn)

	pool, err := database.Connect(context.Background(), dsn)

	if err != nil {
		return nil, err
	}

	log.Info.Println("connected to postgresql database")
	return pool, nil
}

type smtpCfg struct {
	Addr     string
	CA       string
	Admin    string
	Username string
	Password string
}

func (cfg *smtpCfg) connect(log *log.Logger) (*mail.Client, string, error) {
	log.Debug.Println("connecting to smtp addr", cfg.Addr)

	host, _, err := net.SplitHostPort(cfg.Addr)

	if err != nil {
		return nil, "", err
	}

	tlscfg0 := tlsCfg{
		CA: cfg.CA,
	}

	var auth smtp.Auth

	if cfg.Username != "" && cfg.Password != "" {
		log.Debug.Println("using plain auth username =", cfg.Username, "password =", cfg.Password)

		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, host)
	}

	tlscfg, err := tlscfg0.config()

	if err != nil {
		return nil, "", err
	}

	if tlscfg != nil {
		tlscfg.ServerName = host
	}

	cli := &mail.Client{
		Addr:      cfg.Addr,
		Auth:      auth,
		TLSConfig: tlscfg,
	}

	if err := cli.Dial(); err != nil {
		return nil, "", err
	}

	log.Info.Println("connected to smtp server")
	return cli, cfg.Admin, nil
}

type redisCfg struct {
	Addr     string
	Password string
}

func (cfg *redisCfg) connect(log *log.Logger) (*redis.Client, error) {
	redis := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
	})

	log.Debug.Println("connecting to redis database with:", cfg.Addr, cfg.Password)

	if _, err := redis.Ping().Result(); err != nil {
		return nil, err
	}

	log.Info.Println("connected to redis database")
	return redis, nil
}

type storeCfg struct {
	Name  string
	Type  string
	Path  string
	Limit int64
}

func (cfg *storeCfg) store() (fs.FS, int64, error) {
	switch cfg.Type {
	case "file":
		store := fs.New(cfg.Path)

		if cfg.Limit > 0 {
			store = fs.Limit(store, cfg.Limit)
		}
		return store, cfg.Limit, nil
	default:
		return nil, 0, errors.New("unknown store type: " + cfg.Type)
	}
}

type providerCfg struct {
	Secret         string
	Oauth2Endpoint string `config:"oauth2_endpoint"`
	Endpoint       string
	ClientID       string `config:"client_id"`
	ClientSecret   string `config:"client_secret"`
}

func (cfg *providerCfg) auth(name, host string) (auth.Authenticator, error) {
	var config func(host, clientId, clientSecret string, endpoint oauth2.Endpoint) *oauth2.Config

	endpoint := oauth2.Endpoint{
		OAuth: cfg.Oauth2Endpoint,
		API:   cfg.Endpoint,
	}

	switch name {
	case "github":
		config = github.Oauth2Config
	case "gitlab":
		config = gitlab.Oauth2Config
	default:
		return nil, errors.New("config: unknown provider: " + name)
	}
	return oauth2.New(name, nil, config(host, cfg.ClientID, cfg.ClientSecret, endpoint)), nil
}

func (cfg *providerCfg) client(name, host string) (*provider.BaseClient, error) {
	cli := provider.BaseClient{
		Host:     host,
		Secret:   cfg.Secret,
		Endpoint: cfg.Endpoint,
	}

	switch name {
	case "github":
		cli.Init = github.New
	case "gitlab":
		cli.Init = gitlab.New
	default:
		return nil, errors.New("config: unknown provider: " + name)
	}
	return &cli, nil
}
