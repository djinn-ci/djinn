package workerutil

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/worker"
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

	for _, driver := range cfg.Drivers() {
		log.Info.Println("enabled build driver", driver)
	}

	f2, err := os.Open(driverPath)

	if err != nil {
		return nil, cfg, nil, errors.Err(err)
	}

	defer f2.Close()

	drivers, driverCfg, err := config.DecodeDriver(f2.Name(), f2)

	if err != nil {
		return nil, cfg, nil, errors.Err(err)
	}

	return &worker.Worker{
		DB:          db,
		Redis:       redis,
		SMTP:        smtp,
		Admin:       postmaster,
		Block:       cfg.BlockCipher(),
		Log:         log,
		Queue:       cfg.Queue(),
		Parallelism: cfg.Parallelism(),
		Timeout:     cfg.Timeout(),
		Config:      driverCfg,
		Drivers:     drivers,
		Providers:   cfg.Providers(),
		Objects:     cfg.Objects(),
		Artifacts:   cfg.Artifacts(),
	}, cfg, close_, nil
}

func Start(ctx context.Context, w *worker.Worker) {
	go func() {
		if err := w.Run(ctx); err != nil {
			w.Log.Error.Println(errors.Cause(err))
		}
	}()
	w.Log.Info.Println(os.Args[0], "running with", w.Parallelism, "parallelism")
}
