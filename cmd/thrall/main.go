package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/collector"
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/placer"
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

	sigs := make(chan os.Signal)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGKILL)

	r := runner.NewRunner(
		os.Stdout,
		manifest.Env,
		manifest.Objects,
		placer.NewFileSystem(c.Flags.GetString("objects")),
		collector.NewFileSystem(c.Flags.GetString("artifacts")),
		sigs,
	)

	setup := runner.NewStage(setupStage, false)

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

		setup.Add(runner.NewJob(os.Stdout, name, commands, depends, runner.NewPassthrough()))
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

		r.Add(runner.NewStage(name, canFail))
	}

	for _, j := range manifest.Jobs {
		if _, ok := r.Stages[j.Stage]; !ok {
			fmt.Fprintf(r.Writer, "warning: unknown stage %s\n", j.Stage)
		}
	}

	for _, s := range r.Stages {
		jobId := 1

		for _, j := range manifest.Jobs {
			if s.Name != j.Stage {
				continue
			}

			if j.Name == "" {
				j.Name = fmt.Sprintf("%s.%d", j.Stage, jobId)
				jobId++
			}

			s.Add(runner.NewJob(os.Stdout, j.Name, j.Commands, j.Depends, j.Artifacts))
		}
	}

	d, err := driver.NewEnv(os.Stdout, manifest.Driver)

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to configure driver: %s\n", err)
		os.Exit(1)
	}

	stages := c.Flags.GetAll("stage")

	if len(stages) > 0 {
		remove := make([]string, 0, len(r.Stages))

		for runnerStage := range r.Stages {
			keep := false

			for _, flag := range stages {
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

	if err := r.Run(d); err != nil {
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
