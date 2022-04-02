package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"time"

	"djinn-ci.com/build"
	"djinn-ci.com/config"
	"djinn-ci.com/errors"
	"djinn-ci.com/version"
)

func main() {
	var (
		configfile  string
		limit       int64
		showversion bool
	)

	fs := flag.CommandLine
	fs.Int64Var(&limit, "limit", 0, "the limit in bytes after which old artifacts should be removed (deprecated)")
	fs.StringVar(&configfile, "config", "djinn-curator.conf", "the config file to use")
	fs.BoolVar(&showversion, "version", false, "show the version and exit")
	fs.Parse(os.Args[1:])

	if showversion {
		fmt.Printf("%s %s %s/%s\n", os.Args[0], version.Build, runtime.GOOS, runtime.GOARCH)
		return
	}

	if limit > 0 {
		fmt.Fprintln(os.Stderr, "the limit flag has been deprecated in favor of per-user cleanup thresholds")
	}

	f, err := os.Open(configfile)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], errors.Cause(err))
		os.Exit(1)
	}

	defer f.Close()

	cfg, err := config.DecodeCurator(f.Name(), f)

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

		if pidfile != "" {
			if err := os.RemoveAll(pidfile); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
				os.Exit(1)
			}
		}
	}()

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	interval := cfg.Interval()

	t := time.NewTicker(interval)

	artifacts := cfg.Artifacts()

	log.Info.Println(os.Args[0], "started with interval of", interval)

	curator := build.NewCurator(log, db, artifacts)

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

				log.Debug.Println("starting curation")

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
