package workerutil

import (
	"context"
	"flag"
	"fmt"
	"os"

	"djinn-ci.com/config"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
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

func Init(workerPath, driverPath string) (*worker.Worker, *config.Worker, func(), error) {
	env.Load()

	var cfg *config.Worker

	f1, err := os.Open(workerPath)

	if err != nil {
		return nil, cfg, nil, errors.Err(err)
	}

	defer f1.Close()

	cfg, err = config.DecodeWorker(f1.Name(), f1)

	if err != nil {
		return nil, cfg, nil, errors.Err(err)
	}

	pidfile := cfg.Pidfile()

	db := cfg.DB()
	redis := cfg.Redis()
	smtp, postmaster := cfg.SMTP()

	log := cfg.Log()

	close_ := func() {
		db.Close()
		redis.Close()
		smtp.Close()
		log.Close()

		if pidfile != nil {
			if err := os.RemoveAll(pidfile.Name()); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
				os.Exit(1)
			}
		}
	}

	log.Info.Println("consuming from queue:", cfg.Queue())
	log.Info.Println("enabled build driver", cfg.Driver())
	log.Info.Println("using parallelism of:", cfg.Parallelism())

	f2, err := os.Open(driverPath)

	if err != nil {
		return nil, cfg, nil, errors.Err(err)
	}

	defer f2.Close()

	driverName := cfg.Driver()

	driverInit, driverCfg, err := config.DecodeDriver(driverName, f2.Name(), f2)

	if err != nil {
		return nil, cfg, nil, errors.Err(err)
	}

	qerrh := func(err error) {
		log.Error.Println("queue job failed:", err)
	}

	return &worker.Worker{
		DB:        db,
		Redis:     redis,
		SMTP:      smtp,
		Admin:     postmaster,
		Block:     cfg.BlockCipher(),
		Log:       log,
		Consumer:  cfg.Consumer(),
		Queue:     queue.NewMemory(20, qerrh),
		Timeout:   cfg.Timeout(),
		Driver:    driverName,
		Init:      driverInit,
		Config:    driverCfg,
		Providers: cfg.Providers(),
		Objects:   cfg.Objects(),
		Artifacts: cfg.Artifacts(),
	}, cfg, close_, nil
}

func Start(ctx context.Context, w *worker.Worker) {
	go w.Queue.Run(ctx)

	go func() {
		if err := w.Run(ctx); err != nil {
			w.Log.Error.Println(errors.Cause(err))
		}
	}()
}
