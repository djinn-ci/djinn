package main

import (
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"djinn-ci.com/config"
	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/namespace"
	"djinn-ci.com/queue"
	"djinn-ci.com/version"
)

var qname = "jobs"

func main() {
	var (
		cfgfile     string
		showversion bool
	)

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&cfgfile, "config", "djinn-consumer.conf", "the config file to use")
	fs.BoolVar(&showversion, "version", false, "show the version and exit")
	fs.Parse(os.Args[1:])

	if showversion {
		fmt.Printf("%s %s %s/%s\n", os.Args[0], version.Build, runtime.GOOS, runtime.GOARCH)
		return
	}

	f, err := os.Open(cfgfile)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	defer f.Close()

	cfg, err := config.DecodeConsumer(f.Name(), f)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], errors.Cause(err))
		os.Exit(1)
	}

	if pidfile := cfg.Pidfile(); pidfile != "" {
		defer os.RemoveAll(pidfile)
	}

	log := cfg.Log()
	store := cfg.Images()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gob.Register(&image.DownloadJob{})

	db := cfg.DB()

	webhooks := &namespace.WebhookStore{
		Pool:   db,
		AESGCM: cfg.AESGCM(),
	}

	opts := cfg.ConsumerOpts()

	memq := queue.NewMemory(opts.ProcessorConcurrency, func(j queue.Job, err error) {
		log.Error.Println("queue job failed:", j.Name(), err)
	})
	memq.InitFunc("event:images", image.InitEvent(webhooks))

	q := queue.NewRedisConsumer(log, opts)
	q.InitFunc("download_job", image.DownloadJobInit(db, memq, log, store))

	go func() {
		log.Info.Println("consuming jobs from", qname)

		if err := q.Consume(ctx); err != nil {
			log.Error.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	sig := <-c
	log.Info.Println("signal:", sig, "received, shutting down")

	cancel()
}
