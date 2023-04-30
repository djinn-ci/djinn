package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"

	"djinn-ci.com/config"
	"djinn-ci.com/cron"
	"djinn-ci.com/errors"
	"djinn-ci.com/version"
)

func main() {
	var (
		configfile  string
		showversion bool
	)

	fs := flag.CommandLine

	fs.StringVar(&configfile, "config", "djinn-scheduler.conf", "the config file to use")
	fs.BoolVar(&showversion, "version", false, "show the version and exit")
	fs.Parse(os.Args[1:])

	if showversion {
		fmt.Printf("%s %s %s/%s\n", os.Args[0], version.Build, runtime.GOOS, runtime.GOARCH)
		return
	}

	f, err := os.Open(configfile)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	defer f.Close()

	cfg, err := config.DecodeScheduler(f.Name(), f)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], errors.Cause(err))
		os.Exit(1)
	}

	pidfile := cfg.Pidfile()

	db := cfg.DB()
	redis := cfg.Redis()

	log := cfg.Log()

	defer func() {
		db.Close()
		redis.Close()
		log.Close()

		if pidfile != "" {
			if err := os.RemoveAll(pidfile); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
				os.Exit(1)
			}
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	scheduler := cron.NewScheduler(cfg)

	go func() {
		defer func() {
			v := recover()

			if err, ok := v.(error); ok {
				log.Error.Println(err, "\n", string(debug.Stack()))
			}
		}()

		scheduler.Run(ctx, int(cfg.BatchSize()), func(err error) {
			log.Error.Println(err)
		})
	}()

	sig := <-c

	log.Info.Println("signal:", sig, "received, shutting down")
	cancel()
}
