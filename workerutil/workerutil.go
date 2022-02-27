package workerutil

import (
	"context"
	"flag"
	"fmt"
	"os"

	"djinn-ci.com/build"
	"djinn-ci.com/config"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/queue"
	"djinn-ci.com/worker"
)

func ParseFlags(args []string) (string, string, bool) {
	var (
		config  string
		driver  string
		version bool
	)

	fs := flag.NewFlagSet(args[0], flag.ExitOnError)
	fs.StringVar(&config, "config", "djinn-worker.conf", "the config file to use")
	fs.StringVar(&driver, "driver", "djinn-driver.conf", "the driver config to use")
	fs.BoolVar(&version, "version", false, "show the version and exit")
	fs.Parse(args[1:])

	return config, driver, version
}

func Init(workerPath, driverPath string) (*worker.Worker, func(), error) {
	env.Load()

	var cfg *config.Worker

	f1, err := os.Open(workerPath)

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	defer f1.Close()

	cfg, err = config.DecodeWorker(f1.Name(), f1)

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	log := cfg.Log()

	driverName := cfg.Driver()

	if driverName == "os" {
		log.Warn.Println("the os driver should only be used if you trust the builds being submitted, or for testing")
	}

	f2, err := os.Open(driverPath)

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	defer f2.Close()

	driverInit, drivercfg, err := config.DecodeDriver(driverName, f2.Name(), f2)

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	worker := worker.New(cfg, drivercfg, driverInit)

	pidfile := cfg.Pidfile()

	close_ := func() {
		worker.DB.Close()
		worker.Redis.Close()
		worker.SMTP.Close()
		worker.Log.Close()

		if pidfile != "" {
			if err := os.RemoveAll(pidfile); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
				os.Exit(1)
			}
		}
	}

	parallelism := cfg.Parallelism()

	worker.Log.Info.Println("consuming from queue:", cfg.Queue())
	worker.Log.Info.Println("enabled build driver", worker.Driver)
	worker.Log.Info.Println("using parallelism of:", parallelism)

	webhooks := namespace.WebhookStore{
		Pool:   worker.DB,
		AESGCM: worker.AESGCM,
	}

	memq := queue.NewMemory(parallelism, func(j queue.Job, err error) {
		log.Error.Println("queue job failed:", j.Name(), err)
	})
	memq.InitFunc("event:build.started", build.InitEvent(&webhooks))
	memq.InitFunc("event:build.finished", build.InitEvent(&webhooks))

	worker.Queue = memq

	return worker, close_, nil
}

func Start(ctx context.Context, w *worker.Worker) {
	go w.Queue.Consume(ctx)

	go func() {
		if err := w.Run(ctx); err != nil {
			w.Log.Error.Println(errors.Cause(err))
		}
	}()
}
