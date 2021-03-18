package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/andrewpillar/djinn/fs"
	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/manifest"
	"github.com/andrewpillar/djinn/runner"
	"github.com/andrewpillar/djinn/version"
)

var setupStage = "setup"

func exiterr(err error) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
	os.Exit(1)
}

func main() {
	var (
		showversion  bool
		artifactsdir string
		objectsdir   string
		manifestfile string
		driverfile   string
		stage        string
	)

	cfgdir, err := os.UserConfigDir()

	if err != nil {
		cfgdir = "."
	}

	flag := flag.CommandLine

	flag.BoolVar(&showversion, "version", false, "show the version and exit")
	flag.StringVar(&artifactsdir, "artifacts", ".", "the directory to store artifacts")
	flag.StringVar(&objectsdir, "objects", ".", "the directory to place objects from")
	flag.StringVar(&manifestfile, "manifest", ".djinn.yml", "the manifest file to use")
	flag.StringVar(&driverfile, "driver", filepath.Join(cfgdir, "djinn", "driver.toml"), "the driver config to use")
	flag.StringVar(&stage, "stage", "", "the stage to execute")
	flag.Parse(os.Args[1:])

	if showversion {
		fmt.Printf("%s %s %s/%s\n", os.Args[0], version.Build, runtime.GOOS, runtime.GOARCH)
		return
	}

	f1, err := os.Open(manifestfile)

	if err != nil {
		exiterr(err)
	}

	defer f1.Close()

	m, err := manifest.Decode(f1)

	if err != nil {
		exiterr(err)
	}

	if err := m.Validate(); err != nil {
		exiterr(errors.Cause(err))
	}

	f2, err := os.Open(driverfile)

	if err != nil {
		exiterr(err)
	}

	defer f2.Close()

	drivers, driverconf, err := config.DecodeDriver(f2.Name(), f2)

	if err != nil {
		exiterr(errors.Cause(err))
	}

	placer := fs.NewFilesystem(objectsdir)

	if err := placer.Init(); err != nil {
		exiterr(errors.Cause(err))
	}

	collector := fs.NewFilesystem(artifactsdir)

	if err := collector.Init(); err != nil {
		exiterr(errors.Cause(err))
	}

	r := runner.Runner{
		Writer:    os.Stdout,
		Env:       m.Env,
		Objects:   m.Objects,
		Placer:    placer,
		Collector: collector,
	}

	setup := &runner.Stage{
		Name:    setupStage,
		CanFail: false,
	}

	for i, src := range m.Sources {
		name := fmt.Sprintf("clone.%d", i+1)

		commands := []string{
			"git clone " + src.URL + " " + src.Dir,
			"cd " + src.Dir,
		}

		if src.Ref != "" {
			commands = append(commands, "git checkout -q " + src.Ref)
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

	for _, name := range m.Stages {
		canFail := false

		for _, search := range m.AllowFailures {
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

	prev := ""
	jobId := 1

	for _, j := range m.Jobs {
		stage, ok := stages[j.Stage]

		if !ok {
			continue
		}

		if j.Stage != prev {
			jobId = 1
		}

		if j.Name == "" {
			j.Name = fmt.Sprintf("%s.%d", j.Stage, jobId)
			jobId++
		}

		stage.Add(&runner.Job{
			Writer:    os.Stdout,
			Name:      j.Name,
			Commands:  j.Commands,
			Artifacts: j.Artifacts,
		})
	}

	if stage != "" && stage != setupStage {
		r.Remove(stage)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal)

	signal.Notify(sigs, os.Interrupt, os.Kill)

	go func() {
		<-sigs
		cancel()
	}()

	driverInit, err := drivers.Get(m.Driver["type"])

	if err != nil {
		exiterr(err)
	}

	cfg := driverconf[m.Driver["type"]]
	cfg.Merge(m.Driver)

	if err := r.Run(ctx, driverInit(os.Stdout, cfg)); err != nil {
		exiterr(err)
	}
}
