package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/collector"
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/runner"
)

var cloneStage = "clone sources"

func initializeQEMU(build config.Build) runner.Driver {
	arch := build.Driver.Arch

	cpus := os.Getenv("THRALL_QEMU_CPUS")
	memory := os.Getenv("THRALL_QEMU_MEMORY")
	port := os.Getenv("THRALL_QEMU_PORT")
	username := os.Getenv("THRALL_QEMU_USERNAME")
	password := os.Getenv("THRALL_QEMU_PASSWORD")

	timeout, err := strconv.ParseInt(os.Getenv("THRALL_QEMU_TIMEOUT"), 10, 64)

	if err != nil {
		timeout = 10
	}

	if build.Driver.Arch == "" {
		arch = "x86_64"
	}

	if cpus == "" {
		cpus = "2"
	}

	if memory == "" {
		memory = "2048"
	}

	if port == "" {
		port = "2222"
	}

	image := filepath.Join(os.Getenv("THRALL_QEMU_DIR"), build.Driver.Image + ".qcow2")

	return &driver.QEMU{
		Image:    image,
		Arch:     arch,
		CPUs:     cpus,
		Memory:   memory,
		Port:     port,
		Username: username,
		Password: password,
		Timeout:  timeout,
	}
}

func mainCommand(c cli.Command) {
	f, err := os.Open(c.Flags.GetString("config"))

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	build, err := config.DecodeBuild(f)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	dir := c.Flags.GetString("artifacts")
	fs := collector.NewFileSystem(dir)

	r := runner.NewRunner(os.Stdout, fs)

	clone := runner.NewStage(cloneStage, false)

	for i, source := range build.Sources {
		name := fmt.Sprintf("clone.%d", i)

		if source.Ref == "" {
			source.Ref = "master"
		}

		commands := []string{
			"mkdir -p " + source.Dir,
			"git clone " + source.URL + " " + source.Dir,
			"cd " + source.Dir,
			"git checkout -q " + source.Ref,
		}

		depends := []string{}

		if i > 0 {
			depends = []string{fmt.Sprintf("clone.%d", i - 1)}
		}

		clone.Add(runner.NewJob(name, commands, depends, []string{}))
	}

	r.Add(clone)

	for _, name := range build.Stages {
		canFail := false

		for _, search := range build.AllowFailures {
			if name == search {
				canFail = true
				break
			}
		}

		r.Add(runner.NewStage(name, canFail))
	}

	for i, j := range build.Jobs {
		stage, ok := r.Stages[j.Stage]

		if !ok {
			fmt.Fprintf(os.Stderr, "%s: unknown stage %s\n", os.Args[0], j.Stage)
			os.Exit(1)
		}

		if j.Name == "" {
			j.Name = fmt.Sprintf("%s.%d", stage.Name, i + 1)
		}

		stage.Add(runner.NewJob(j.Name, j.Commands, j.Depends, j.Artifacts))
	}

	var d runner.Driver

	switch build.Driver.Type {
		case "docker":
			d = driver.NewDocker(build.Driver.Image, build.Driver.Workspace)
		case "qemu":
			d = initializeQEMU(build)
		default:
			fmt.Fprintf(os.Stderr, "%s: unknown driver %s\n", os.Args[0], build.Driver.Type)
			os.Exit(1)
	}

	stage := c.Flags.GetString("stage")

	if stage != "" {
		for name := range r.Stages {
			if name == stage || name == cloneStage {
				continue
			}

			r.Remove(name)
		}
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

	cmd := c.Main(mainCommand)

	cmd.AddFlag(&cli.Flag{
		Name:     "artifacts",
		Short:    "-a",
		Long:     "--artifacts",
		Argument: true,
		Default:  ".",
	})

	cmd.AddFlag(&cli.Flag{
		Name:     "config",
		Short:    "-c",
		Long:     "--config",
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
