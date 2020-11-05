package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"time"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/version"
)

func main() {
	var (
		configfile  string
		limit       int64
		showversion bool
	)

	fs := flag.CommandLine
	fs.Int64Var(&limit, "limit", 1<<30, "remove artifacts that go over this limit")
	fs.StringVar(&configfile, "config", "djinn-curator.toml", "the config file to use")
	fs.BoolVar(&showversion, "version", false, "show the version and exit")
	fs.Parse(os.Args[1:])

	if showversion {
		fmt.Printf("%s %s %s\n", os.Args[0], version.Tag, version.Ref)
		return
	}

	f, err := os.Open(configfile)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], errors.Cause(err))
		os.Exit(1)
	}

	defer f.Close()

	cfg, err := config.DecodeCurator(f)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], errors.Cause(err))
		os.Exit(1)
	}

	pidfile := cfg.Pidfile()

	db := cfg.DB()

	log := cfg.Log()

	defer func() {
		db.Close()
		log.Close()

		if pidfile != nil {
			if err := os.RemoveAll(pidfile.Name()); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
				os.Exit(1)
			}
		}
	}()

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	t := time.NewTicker(time.Minute)

	artifacts := cfg.Artifacts()

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

				curator := build.NewCurator(db, artifacts, limit)

				if err := curator.Invoke(); err != nil {
					log.Error.Println(err)
				}
			}()
		case sig := <-c:
			log.Info.Println("signal:", sig, "received, shutting down")
			break loop
		}
	}
}
