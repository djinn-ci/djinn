package main

import (
	"context"
	"flag"
	"fmt"
	"os"

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := queue.NewCurlyQ(nil, con)
	q.InitFunc("download_job", image.DownloadJobInit(db, store))

	go func() {
		if err := q.Consume(ctx); err != nil {
			// log error
		}
	}()

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	sig := <-c

	cancel()
}
