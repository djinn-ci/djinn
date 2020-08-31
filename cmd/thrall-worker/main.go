package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/block"
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/driver/docker"
	"github.com/andrewpillar/thrall/driver/qemu"
	"github.com/andrewpillar/thrall/driver/ssh"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"

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
)

func main() {
	var (
		showversion bool
		configfile  string
		driverfile  string
	)

	flag.BoolVar(&showversion, "version", false, "show the version and exit")
	flag.StringVar(&configfile, "config", "thrall-worker.toml", "the config file to use")
	flag.StringVar(&driverfile, "driver", "thrall-driver.toml", "the driver config to use")
	flag.Parse()

	if showversion {
		fmt.Println("thrall-worker", Version, Build)
		os.Exit(0)
	}

	log := log.New(os.Stdout)

	cf, err := os.Open(configfile)

	if err != nil {
		log.Error.Fatalf("failed to open worker config: %s\n", err)
	}

	defer cf.Close()

	df, err := os.Open(driverfile)

	if err != nil {
		log.Error.Fatalf("failed to open driver config: %s\n", err)
	}

	defer df.Close()

	cfg, err := config.DecodeWorker(cf)

	if err != nil {
		log.Error.Fatalf("failed to decode worker config: %s\n", err)
	}

	if cfg.Pidfile != "" {
		pidf, err := os.OpenFile(cfg.Pidfile, os.O_WRONLY|os.O_CREATE, 0660)

		if err != nil {
			log.Error.Fatalf("failed to create pidfile: %s\n", err)
		}

		pidf.Write([]byte(strconv.FormatInt(int64(os.Getpid()), 10)))
		pidf.Close()
	}

	tree, err := toml.LoadReader(df)

	if err != nil {
		log.Error.Fatalf("failed to load driver config: %s\n", err)
	}

	if err := config.ValidateDrivers(tree); err != nil {
		log.Error.Fatalf("driver config validation failed: %s\n", err)
	}

	drivers := driver.NewRegistry()

	for _, name := range tree.Keys() {
		drivers.Register(name, driverInits[name])
	}

	if cfg.Queue == "" {
		log.Error.Fatalf("no queue to work from\n")
	}

	log.SetLevel(cfg.Log.Level)

	logf, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		log.Error.Fatalf("failed to open log file %s: %s\n", cfg.Log.File, err)
	}

	defer logf.Close()

	log.SetWriter(logf)

	blockCipher, err := crypto.NewBlock([]byte(cfg.Crypto.Block))

	if err != nil {
		log.Error.Fatalf("failed to setup block cipher: %s\n", errors.Cause(err))
	}

	host, port, err := net.SplitHostPort(cfg.Database.Addr)

	if err != nil {
		log.Error.Fatal(err)
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
		log.Error.Fatalf("failed to connect to database: %s\n", errors.Cause(err))
	}

	log.Info.Println("connected to postgresql database")

	redis := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})

	if _, err := redis.Ping().Result(); err != nil {
		log.Error.Fatalf("failed to ping redis: %s\n", err)
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
		log.Error.Fatalf("failed to setup queue %s: %s\n", cfg.Queue, err)
	}

	images := blockstores[cfg.Images.Type](cfg.Images.Path, cfg.Images.Limit)
	objects := blockstores[cfg.Objects.Type](cfg.Objects.Path, cfg.Objects.Limit)
	artifacts := blockstores[cfg.Artifacts.Type](cfg.Artifacts.Path, cfg.Artifacts.Limit)

	if err := images.Init(); err != nil {
		log.Error.Fatalf("failed to initialize image store: %s\n", errors.Cause(err))
	}

	if err := objects.Init(); err != nil {
		log.Error.Fatalf("failed to initialize object store: %s\n", errors.Cause(err))
	}

	if err := artifacts.Init(); err != nil {
		log.Error.Fatalf("failed to initialize artifact store: %s\n", errors.Cause(err))
	}

	timeout, err := time.ParseDuration(cfg.Timeout)

	if err != nil {
		log.Error.Fatalf("failed to parse worker timeout: %s\n", err)
	}

	driverconf := make(map[string]map[string]interface{})

	for _, key := range tree.Keys() {
		subtree := tree.Get(key).(*toml.Tree)

		driverconf[key] = subtree.ToMap()
	}

	var client *smtp.Client

	if cfg.SMTP.Addr != "" {
		var auth smtp.Auth

		host, _, err := net.SplitHostPort(cfg.SMTP.Addr)

		if cfg.SMTP.Username != "" && cfg.SMTP.Password != "" {
			if err != nil {
				log.Error.Fatalf("invalid smtp address: %s\n", err)
			}

			auth = smtp.PlainAuth("", cfg.SMTP.Username, cfg.SMTP.Password, host)
		}

		var tlscfg *tls.Config

		if cfg.SMTP.CA != "" {
			b, err := ioutil.ReadFile(cfg.SMTP.CA)

			if err != nil {
				log.Error.Fatalf("failed to read ca file: %s\n", err)
			}

			pool := x509.NewCertPool()

			if !pool.AppendCertsFromPEM(b) {
				log.Error.Fatalf("failed to append certificates from PEM, please check if valid\n")
			}

			log.Debug.Println("loaded root ca chain")

			tlscfg = &tls.Config{
				ServerName: host,
				RootCAs:    pool,
			}
		}

		client, err = smtp.Dial(cfg.SMTP.Addr)

		if err != nil {
			log.Error.Fatalf("failed to create smtp client: %s\n", err)
		}

		log.Debug.Println("connected to smtp server", cfg.SMTP.Addr)

		if tlscfg != nil {
			log.Debug.Println("issuing STARTTLS command to smtp server")

			if err := client.StartTLS(tlscfg); err != nil {
				log.Error.Fatalf("STARTTLS command failed: %s\n", err)
			}
		}

		if auth != nil {
			if err := client.Auth(auth); err != nil {
				log.Error.Fatalf("smtp client auth failed: %s\n", err)
			}
		}

		log.Info.Println("connected to smtp server", cfg.SMTP.Addr)
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
		timeout:    timeout,
		server:     queue,
		placer:     objects,
		collector:  artifacts,
	}

	w.init(cfg.Queue, cfg.Parallelism)

	if err := w.worker.Launch(); err != nil {
		log.Error.Fatalf("failed to launch worker: %s\n", errors.Cause(err))
	}

	if cfg.Pidfile != "" {
		if err := os.RemoveAll(cfg.Pidfile); err != nil {
			log.Error.Println("failed to remove pidfile", err)
		}
	}
}
