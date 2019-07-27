package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/runner"
)

var (
	setupStage = "setup"

	Build string
)

func mainCommand(c cli.Command) {
	f, err := os.Open(c.Flags.GetString("manifest"))

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	manifest, err := config.DecodeManifest(f)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	if err := manifest.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	r := runner.Runner{
		Writer:    os.Stdout,
		Env:       manifest.Env,
		Objects:   manifest.Objects,
		Placer:    filestore.NewFileSystem(c.Flags.GetString("objects"), 0),
		Collector: filestore.NewFileSystem(c.Flags.GetString("artifacts"), 0),
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

		depends := []string{"create driver"}

		if i > 0 {
			depends = append(depends, fmt.Sprintf("clone.%d", i))
		}

		setup.Add(&runner.Job{
			Writer:    os.Stdout,
			Name:      name,
			Commands:  commands,
			Depends:   depends,
			Artifacts: runner.NewPassthrough(),
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
				Depends:   j.Depends,
				Artifacts: j.Artifacts,
			})
		}
	}

	d, err := driver.NewEnv(os.Stdout, manifest.Driver)

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

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGKILL)

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
			fmt.Println("thrall", Build)
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
