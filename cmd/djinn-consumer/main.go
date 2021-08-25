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
	"djinn-ci.com/queue"
	"djinn-ci.com/version"
)

var validqueues = map[string]struct{}{
	"image_downloads": {},
}

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

	qname := cfg.QueueName()

	if _, ok := validqueues[qname]; !ok {
		valid := make([]string, 0, len(validqueues))

		for qname := range validqueues {
			valid = append(valid, qname)
		}

		fmt.Fprintf(os.Stderr, "%s: invalid queue to consume from, must be one of: %v\n", os.Args[0], valid)
		os.Exit(1)
	}

	if pidfile := cfg.Pidfile(); pidfile != nil {
		defer os.RemoveAll(pidfile.Name())
	}

	log := cfg.Log()

	store, ok := cfg.Store("images")

	if !ok {
		fmt.Fprintf(os.Stderr, "%s: image store not defined\n", os.Args[0])
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gob.Register(&image.DownloadJob{})

	q := queue.NewCurlyQ(log, nil, cfg.Consumer())
	q.InitFunc("download_job", image.DownloadJobInit(cfg.DB(), cfg.Log(), store))

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
