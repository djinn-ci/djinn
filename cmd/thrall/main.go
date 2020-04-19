package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/driver/ssh"
	"github.com/andrewpillar/thrall/driver/qemu"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/runner"
)

var (
	setupStage = "setup"

	Build   string
	Version string
)

func qemuRealpath(dir string) func(string, string) (string, error) {
	return func(arch, name string) (string, error) {
		path := filepath.Join(dir, arch, filepath.Join(strings.Split(name, "/")...))
		info, err := os.Stat(path)

		if err != nil {
			return "", err
		}
		if info.IsDir() {
			return "", errors.New("image is not a file")
		}
		return path, nil
	}
}

func configureDrivers(driver config.Driver, manifest config.Manifest) {
	runner.ConfigureDriver(
		"qemu",
		qemu.Configure(
			qemu.Key(driver.QEMU.Key),
			qemu.CPUs(driver.QEMU.CPUs),
			qemu.Memory(driver.QEMU.Memory),
			qemu.Image(manifest.Driver["image"]),
			qemu.Realpath(qemuRealpath(driver.QEMU.Disks)),
		),
	)

	runner.ConfigureDriver(
		"ssh",
		ssh.Configure(
			ssh.Key(driver.SSH.Key),
			ssh.Timeout(time.Duration(time.Second*time.Duration(driver.SSH.Timeout))),
		),
	)
}

func mainCommand(c cli.Command) {
	mf, err := os.Open(c.Flags.GetString("manifest"))

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	defer mf.Close()

	manifest, err := config.DecodeManifest(mf)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	if err := manifest.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	df, err := os.Open(c.Flags.GetString("driver"))

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	defer df.Close()

	driverCfg, err := config.DecodeDriver(df)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	configureDrivers(driverCfg, manifest)

	placer, err := filestore.NewFileSystem(config.Storage{
		Kind: "file",
		Path: c.Flags.GetString("objects"),
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	collector, err := filestore.NewFileSystem(config.Storage{
		Kind: "file",
		Path: c.Flags.GetString("artifacts"),
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	r := runner.Runner{
		Writer:    os.Stdout,
		Env:       manifest.Env,
		Objects:   manifest.Objects,
		Placer:    placer,
		Collector: collector,
	}

	setup := &runner.Stage{
		Name:    setupStage,
		CanFail: false,
	}

	for i, src := range manifest.Sources {
		name := fmt.Sprintf("clone.%d", i + 1)

		commands := []string{
			"git clone " + src.URL + " " + src.Dir,
			"cd " + src.Dir,
			"git checkout -q " + src.Ref,
		}

		if src.Dir != "" {
			commands = append([]string{"mkdir -p " + src.Dir}, commands...)
		}

		setup.Add(&runner.Job{
			Writer:    os.Stdout,
			Name:      name,
			Commands:  commands,
			Artifacts: runner.Passthrough{},
		})
	}

	r.Add(setup)

	for _, name := range manifest.Stages {
		canFail := false

		for _, search := range manifest.AllowFailures {
			if name == search {
				canFail = true
				break
			}
		}

		r.Add(&runner.Stage{
			Name:    name,
			CanFail: canFail,
		})
	}

	stages := r.Stages()

	for _, s := range stages {
		jobId := 1

		for _, j := range manifest.Jobs {
			if s.Name != j.Stage {
				continue
			}

			if j.Name == "" {
				j.Name = fmt.Sprintf("%s.%d", j.Stage, jobId)
				jobId++
			}

			s.Add(&runner.Job{
				Writer:    os.Stdout,
				Name:      j.Name,
				Commands:  j.Commands,
				Artifacts: j.Artifacts,
			})
		}
	}

	configure, err := runner.GetDriver(manifest.Driver["type"])

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get driver: %s\n", err)
		os.Exit(1)
	}

	d, err := configure(os.Stdout)

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to configure driver: %s\n", err)
		os.Exit(1)
	}

	only := c.Flags.GetAll("stage")

	if len(only) > 0 {
		remove := make([]string, 0, len(stages))

		for runnerStage := range stages {
			keep := false

			for _, flag := range only {
				if runnerStage == flag.GetString() || runnerStage == setupStage {
					keep = true
				}
			}

			if !keep {
				remove = append(remove, runnerStage)
			}
		}

		r.Remove(remove...)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal)

	signal.Notify(sigs, os.Interrupt, os.Kill)

	go func() {
		<-sigs
		cancel()
	}()

	if err := r.Run(ctx, d); err != nil {
		os.Exit(1)
	}
}

func main() {
	c := cli.New()

	c.AddFlag(&cli.Flag{
		Name:      "help",
		Long:      "--help",
		Exclusive: true,
		Handler:   func(f cli.Flag, c cli.Command) {
			fmt.Println(usage)
		},
	})

	c.AddFlag(&cli.Flag{
		Name:      "version",
		Long:      "--version",
		Exclusive: true,
		Handler:   func(f cli.Flag, c cli.Command) {
			fmt.Println("thrall", Build, Version)
		},
	})

	cmd := c.MainCommand(mainCommand)

	cmd.AddFlag(&cli.Flag{
		Name:     "artifacts",
		Short:    "-a",
		Long:     "--artifacts",
		Argument: true,
		Default:  ".",
	})

	cmd.AddFlag(&cli.Flag{
		Name:     "objects",
		Short:    "-o",
		Long:     "--objects",
		Argument: true,
		Default:  ".",
	})

	cmd.AddFlag(&cli.Flag{
		Name:     "manifest",
		Short:    "-m",
		Long:     "--manifest",
		Argument: true,
		Default:  ".thrall.yml",
	})

	cmd.AddFlag(&cli.Flag{
		Name:     "driver",
		Short:    "-d",
		Long:     "--driver",
		Argument: true,
		Default:  "thrall-driver.toml",
	})

	cmd.AddFlag(&cli.Flag{
		Name:     "stage",
		Short:    "-s",
		Long:     "--stage",
		Argument: true,
	})

	if err := c.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
