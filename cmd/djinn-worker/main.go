package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/errors"
)

var (
	Version string
	Build   string
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

	fcfg, err := os.Open(configfile)

	if err != nil {
		return err
	}

	defer fcfg.Close()

	fdriver, err := os.Open(driverfile)

	if err != nil {
		return err
	}

	defer fdriver.Close()

	cfg, err := config.DecodeWorker(fcfg)

	if err != nil {
		return err
	}

	drivers, driverconf, err := config.DecodeDriver(fdriver)

	pidfile := cfg.Pidfile()

	db := cfg.DB()
	redis := cfg.Redis()
	smtp, postmaster := cfg.SMTP()

	log := cfg.Log()

	defer db.Close()
	defer redis.Close()
	defer smtp.Close()
	defer log.Close()

	queue := cfg.Queue()

	w := worker{
		db:         db,
		redis:      redis,
		block:      cfg.BlockCipher(),
		log:        log,
		driverconf: driverconf,
		drivers:    drivers,
		providers:  cfg.Providers(),
		timeout:    cfg.Timeout(),
		server:     queue,
		placer:     cfg.Objects(),
		collector:  cfg.Artifacts(),
	}

	w.smtp.client = smtp
	w.smtp.from = postmaster

	w.init(queue.GetConfig().DefaultQueue, cfg.Parallelism())

	if err := w.worker.Launch(); err != nil {
		return err
	}

	if pidfile != nil {
		if err := os.RemoveAll(pidfile.Name()); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := run(os.Stdout, os.Stderr, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", errors.Cause(err))
		os.Exit(1)
	}
}
