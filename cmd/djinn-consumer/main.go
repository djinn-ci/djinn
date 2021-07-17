package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"djinn-ci.com/config"
	"djinn-ci.com/image"
	"djinn-ci.com/queue"
)

func main() {
	var (
		cfgfile string
	)

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&cfgfile, "config", "djinn-consumer.conf", "the config file to use")
	fs.Parse(os.Args[1:])

	f, err := os.Open(cfgfile)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	defer f.Close()

	cfg, err := config.DecodeConsumer(f.Name(), f)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
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

	q := queue.NewCurlyQ(nil, cfg.Consumer())
	q.InitFunc("download_job", image.DownloadJobInit(cfg.DB(), store))

	go func() {
		log.Info.Println("consuming jobs from", cfg.QueueName())

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
