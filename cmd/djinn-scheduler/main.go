package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/cron"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/log"

	"github.com/RichardKnop/machinery/v1"
	qconfig "github.com/RichardKnop/machinery/v1/config"
)

var (
	Build   string
	Version string
)

func runBatches(log *log.Logger, queues map[string]*machinery.Server, crons *cron.Store, builds *build.Store) (err error) {
	defer func() {
		v := recover()

		if e, ok := v.(error); ok {
			err = errors.New(e.Error() + "\n" + string(debug.Stack()))
		}
	}()

	batcher := cron.NewBatcher(crons, 1000)

	for batcher.Next() {
		cc := batcher.Crons()

		log.Debug.Println("invoking", len(cc), "cron job(s)")

		for _, c := range cc {
			b, err := crons.Invoke(c)

			if err != nil {
				log.Error.Println("failed to invoke cron", errors.Err(err))
				continue
			}

			queue, ok := queues[b.Manifest.Driver["type"]]

			if !ok {
				log.Error.Println("invalid build driver", b.Manifest.Driver["type"], "for build", b.ID)
				continue
			}

			log.Debug.Println("submitting build", b.ID, "for cron", c.ID)

			if err := builds.Submit(queue, "djinn-scheduler", b); err != nil {
				log.Error.Println("failed to submit build", errors.Err(err))
			}
		}
	}
	return errors.Err(batcher.Err())
}

func run(stdout, stderr io.Writer, args []string) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	var (
		showversion bool
		configfile  string
	)

	flags.BoolVar(&showversion, "version", false, "show the version and exit")
	flags.StringVar(&configfile, "config", "djinn-scheduler.toml", "the config file to use")
	flags.Parse(args[1:])

	if showversion {
		fmt.Fprintf(stdout, "%s %s %s\n", args[0], Version, Build)
		return nil
	}

	log := log.New(stdout)

	f, err := os.Open(configfile)

	if err != nil {
		return err
	}

	defer f.Close()

	cfg, err := config.DecodeScheduler(f)

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

	log.SetLevel(cfg.Log.Level)

	logf, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		return err
	}

	defer logf.Close()

	log.Info.Println("logging initialized, writing to", logf.Name())

	log.SetWriter(logf)

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

	defer db.Close()

	log.Info.Println("connected to postgresql database")

	broker := "redis://"

	if cfg.Redis.Password != "" {
		broker += cfg.Redis.Password + "@"
	}

	broker += cfg.Redis.Addr

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
			if err := runBatches(log, queues, crons, builds); err != nil {
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
