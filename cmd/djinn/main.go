package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/driver"
	"github.com/andrewpillar/djinn/driver/docker"
	"github.com/andrewpillar/djinn/driver/qemu"
	"github.com/andrewpillar/djinn/driver/ssh"
	"github.com/andrewpillar/djinn/runner"

	"github.com/pelletier/go-toml"
)

var (
	setupStage = "setup"

	Build   string
	Version string

	driverInits = map[string]driver.Init{
		"docker": docker.Init,
		"ssh":    ssh.Init,
		"qemu":   qemu.Init,
	}
)

func run(stdout, stderr io.Writer, args []string) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

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

	flags.BoolVar(&showversion, "version", false, "show the version and exit")
	flags.StringVar(&artifactsdir, "artifacts", ".", "the directory to store artifacts")
	flags.StringVar(&objectsdir, "objects", ".", "the directory to place objects from")
	flags.StringVar(&manifestfile, "manifest", ".djinn.yml", "the manifest file to use")
	flags.StringVar(&driverfile, "driver", filepath.Join(cfgdir, "djinn", "driver.toml"), "the driver config to use")
	flags.StringVar(&stage, "stage", "", "the stage to execute")
	flags.Parse(args[1:])

	if showversion {
		fmt.Fprintf(stdout, "%s %s %s\n", args[0], Version, Build)
		return nil
	}

	mf, err := os.Open(manifestfile)

	if err != nil {
		return err
	}

	defer mf.Close()

	manifest, err := config.DecodeManifest(mf)

	if err != nil {
		return err
	}

	if err := manifest.Validate(); err != nil {
		return err
	}

	df, err := os.Open(driverfile)

	if err != nil {
		return err
	}

	defer df.Close()

	tree, err := toml.LoadReader(df)

	if err != nil {
		return err
	}

	if err := config.ValidateDrivers(driverfile, tree); err != nil {
		return err
	}

	drivers := driver.NewRegistry()

	for _, name := range tree.Keys() {
		drivers.Register(name, driverInits[name])
	}

	placer := block.NewFilesystem(objectsdir)

	if err := placer.Init(); err != nil {
		return err
	}

	collector := block.NewFilesystem(artifactsdir)

	if err := collector.Init(); err != nil {
		return err
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
		name := fmt.Sprintf("clone.%d", i+1)

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

	prev := ""
	jobId := 1

	for _, j := range manifest.Jobs {
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

	driverInit, err := drivers.Get(manifest.Driver["type"])

	if err != nil {
		return err
	}

	merged := make(map[string]interface{})

	for _, key := range tree.Keys() {
		tree := tree.Get(key).(*toml.Tree)

		for k, v := range tree.ToMap() {
			merged[k] = v
		}
	}

	for k, v := range manifest.Driver {
		merged[k] = v
	}

	d := driverInit(os.Stdout, merged)

	return r.Run(ctx, d)
}

func main() {
	if err := run(os.Stdout, os.Stderr, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
