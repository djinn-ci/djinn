package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/cron"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/log"

	"github.com/go-redis/redis"

	"github.com/RichardKnop/machinery/v1"
	qconfig "github.com/RichardKnop/machinery/v1/config"
)

var (
	Build   string
	Version string
)

func runBatches(host string, queues map[string]*machinery.Server, crons *cron.Store, builds *build.Store) error {
	batcher := cron.NewBatcher(crons, 1000)
	errs := make([]error, 0)

	for batcher.Next() {
		cc := batcher.Crons()

		for _, c := range cc {
			b, err := crons.Invoke(c)

			if err != nil {
				errs = append(errs, err)
				continue
			}

			queue, ok := queues[b.Manifest.Driver["Type"]]

			if !ok {
				continue
			}

			if err := builds.Submit(queue, host, b); err != nil {
				errs = append(errs, err)
				continue
			}
		}
	}

	if err := batcher.Err(); err != nil {
		return errors.Err(err)
	}
	return errors.Slice(errs).Err()
}

func run(stdout, stderr io.Writer, args []string) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	var (
		showversion bool
		configfile  string
	)

	flags.BoolVar(&showversion, "version", false, "show the version and exit")
	flags.StringVar(&configfile, "config", "djinn-server.toml", "the config file to use")
	flags.Parse(args[1:])

	if showversion {
		fmt.Fprintf(stdout, "%s %s %s\n", os.Args[0], Version, Build)
		return nil
	}

	log := log.New(stdout)

	f, err := os.Open(configfile)

	if err != nil {
		return err
	}

	defer f.Close()

	cfg, err := config.DecodeServer(f)

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

	log.Debug.Println("connecting to redis database with:", cfg.Redis.Addr, cfg.Redis.Password)

	if _, err := redis.Ping().Result(); err != nil {
		return err
	}

	defer redis.Close()

	log.Info.Println("connected to redis database")

	broker := "redis://"

	if cfg.Redis.Password != "" {
		broker += cfg.Redis.Password + "@"
	}

	queues := make(map[string]*machinery.Server)

	for _, d := range cfg.Drivers {
		queue, err := machinery.NewServer(&qconfig.Config{
			Broker:        broker,
			DefaultQueue:  d.Queue,
			ResultBackend: broker,
		})

		if err != nil {
			return err
		}
		queues[d.Type] = queue
	}

	broker += cfg.Redis.Addr

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	t := time.NewTicker(time.Second)

	crons := cron.NewStore(db)
	builds := build.NewStore(db)

loop:
	for {
		select {
		case <-t.C:
			println("running batches", time.Now().Format("Mon, 2 Jan 15:04 2006"))
			if err := runBatches(cfg.Host, queues, crons, builds); err != nil {
				log.Error.Println("failed to run cron job batch", errors.Err(err))
			}
		case sig := <-c:
			log.Info.Println("signal:", sig, "received, shutting down")
			break loop
		}
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
