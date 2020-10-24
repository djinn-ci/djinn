package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"time"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/driver"
	"github.com/andrewpillar/djinn/driver/docker"
	"github.com/andrewpillar/djinn/driver/qemu"
	"github.com/andrewpillar/djinn/driver/ssh"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/log"
	"github.com/andrewpillar/djinn/mail"
	"github.com/andrewpillar/djinn/provider"
	"github.com/andrewpillar/djinn/provider/github"
	"github.com/andrewpillar/djinn/provider/gitlab"

	"github.com/go-redis/redis"

	"github.com/pelletier/go-toml"

	"github.com/RichardKnop/machinery/v1"
	qconfig "github.com/RichardKnop/machinery/v1/config"
)

var (
	Version string
	Build   string

	driverInits = map[string]driver.Init{
		"docker": docker.Init,
		"ssh":    ssh.Init,
		"qemu":   qemu.Init,
	}

	blockstores = map[string]func(string, int64) block.Store{
		"file": func(dsn string, limit int64) block.Store {
			return block.NewFilesystemWithLimit(dsn, limit)
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

func run(stdout, stderr io.Writer, args []string) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	var (
		showversion bool
		configfile  string
		driverfile  string
	)

	flags.BoolVar(&showversion, "version", false, "show the version and exit")
	flags.StringVar(&configfile, "config", "djinn-worker.toml", "the config file to use")
	flags.StringVar(&driverfile, "driver", "djinn-driver.toml", "the driver config to use")
	flags.Parse(args[1:])

	if showversion {
		fmt.Fprintf(stdout, "%s %s %s\n", args[0], Version, Build)
		return nil
	}

	log := log.New(stdout)

	cf, err := os.Open(configfile)

	if err != nil {
		return err
	}

	defer cf.Close()

	df, err := os.Open(driverfile)

	if err != nil {
		return err
	}

	defer df.Close()

	cfg, err := config.DecodeWorker(cf)

	if err != nil {
		return err
	}

	if cfg.Pidfile != "" {
		pidf, err := os.OpenFile(cfg.Pidfile, os.O_WRONLY|os.O_CREATE, 0660)

		if err != nil {
			return err
		}

		pidf.Write([]byte(strconv.FormatInt(int64(os.Getpid()), 10)))
		pidf.Close()
	}

	tree, err := toml.LoadReader(df)

	if err != nil {
		return err
	}

	if err := config.ValidateDrivers(driverfile, tree); err != nil {
		return err
	}

	drivers := driver.NewRegistry()

	for _, name := range tree.Keys() {
		drivers.Register(name, driverInits[name])
	}

	if cfg.Queue == "" {
		return errors.New("no queue to work from")
	}

	log.SetLevel(cfg.Log.Level)

	logf, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		return err
	}

	defer logf.Close()

	log.SetWriter(logf)

	blockCipher, err := crypto.NewBlock([]byte(cfg.Crypto.Block))

	if err != nil {
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

	queue, err := machinery.NewServer(&qconfig.Config{
		Broker:        broker,
		DefaultQueue:  cfg.Queue,
		ResultBackend: broker,
	})

	if err != nil {
		return err
	}

	images := blockstores[cfg.Images.Type](cfg.Images.Path, cfg.Images.Limit)
	objects := blockstores[cfg.Objects.Type](cfg.Objects.Path, cfg.Objects.Limit)
	artifacts := blockstores[cfg.Artifacts.Type](cfg.Artifacts.Path, cfg.Artifacts.Limit)

	if err := images.Init(); err != nil {
		return err
	}

	if err := objects.Init(); err != nil {
		return err
	}

	if err := artifacts.Init(); err != nil {
		return err
	}

	providers := provider.NewRegistry()

	for _, p := range cfg.Providers {
		factory, ok := providerFactories[p.Name]

		if !ok {
			log.Error.Fatalf("unknown provider: %s\n", p.Name)
		}
		providers.Register(
			p.Name,
			factory("", p.Endpoint, p.Secret, p.ClientID, p.ClientSecret),
		)
	}

	timeout, err := time.ParseDuration(cfg.Timeout)

	if err != nil {
		return err
	}

	driverconf := make(map[string]map[string]interface{})

	for _, key := range tree.Keys() {
		subtree := tree.Get(key).(*toml.Tree)

		driverconf[key] = subtree.ToMap()
	}

	var client *smtp.Client

	if cfg.SMTP.Addr != "" {
		log.Debug.Println("connecting to smtp addr", cfg.SMTP.Addr)

		if cfg.SMTP.Username != "" && cfg.SMTP.Password != "" {
			log.Debug.Println("connecting to smtp with plain auth username =", cfg.SMTP.Username, "password =", cfg.SMTP.Password)
		}

		if cfg.SMTP.CA != "" {
			log.Debug.Println("connecting to smtp with tls")
		}

		client, err = mail.NewClient(mail.ClientConfig{
			CA:       cfg.SMTP.CA,
			Addr:     cfg.SMTP.Addr,
			Username: cfg.SMTP.Username,
			Password: cfg.SMTP.Password,
		})

		if err != nil {
			log.Error.Fatalf("failed to connect to smtp server: %s\n", errors.Cause(err))
		}

		log.Info.Println("connected to smtp server")
	}

	w := worker{
		db:    db,
		redis: redis,
		smtp: struct {
			client *smtp.Client
			from   string
		}{
			client: client,
			from:   cfg.SMTP.Admin,
		},
		block:      blockCipher,
		log:        log,
		driverconf: driverconf,
		drivers:    drivers,
		providers:  providers,
		timeout:    timeout,
		server:     queue,
		placer:     objects,
		collector:  artifacts,
	}

	w.init(cfg.Queue, cfg.Parallelism)

	if err := w.worker.Launch(); err != nil {
		return err
	}

	if cfg.Pidfile != "" {
		if err := os.RemoveAll(cfg.Pidfile); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := run(os.Stdout, os.Stderr, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
