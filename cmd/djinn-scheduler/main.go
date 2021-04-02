package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/cron"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/version"
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
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], errors.Cause(err))
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

		if pidfile != nil {
			if err := os.RemoveAll(pidfile.Name()); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
				os.Exit(1)
			}
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	d := cfg.Interval()
	t := time.NewTicker(d)

	producers := cfg.Producers()
	batchsize := cfg.BatchSize()

	hasher := cfg.Hasher()

loop:
	for {
		select {
		case <-t.C:
			func() {
				defer func() {
					v := recover()

					if e, ok := v.(error); ok {
						log.Error.Println(e.Error() + "\n" + string(debug.Stack()))
					}
				}()

				batcher := cron.NewBatcher(db, hasher, batchsize, func(err error) {
					log.Error.Println(err)
				})

				log.Debug.Println("loading batch of size", batchsize)

				for batcher.Load() {
					log.Debug.Println("scheduled", len(batcher.Batch()), "cron job(s)")

					n := batcher.Invoke(ctx, producers)

					log.Debug.Println("submitted", n, "build(s)")
				}

				if err := batcher.Err(); err != nil {
					log.Error.Println("batch error", err)
				}
			}()
		case sig := <-c:
			t.Stop()
			cancel()
			log.Info.Println("signal:", sig, "received, shutting down")
			break loop
		}
	}
}
