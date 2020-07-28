package main

import (
	"context"
	"fmt"
	"flag"
	"os"
	"os/signal"

	"github.com/andrewpillar/thrall/block"
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/driver/docker"
	"github.com/andrewpillar/thrall/driver/qemu"
	"github.com/andrewpillar/thrall/driver/ssh"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

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

func main() {
	var (
		showversion  bool
		artifactsdir string
		objectsdir   string
		manifestfile string
		driverfile   string
		stage        string
	)

	flag.BoolVar(&showversion, "version", false, "show the version and exit")
	flag.StringVar(&artifactsdir, "artifacts", ".", "the directory to store artifacts")
	flag.StringVar(&objectsdir, "objects", ".", "the directory to place objects from")
	flag.StringVar(&manifestfile, "manifest", ".thrall.yml", "the manifest file to use")
	flag.StringVar(&driverfile, "driver", "thrall-driver.toml", "the driver config to use")
	flag.StringVar(&stage, "stage", "", "the stage to execute")
	flag.Parse()

	if showversion {
		fmt.Println("thrall", Version, Build)
		os.Exit(0)
	}

	mf, err := os.Open(manifestfile)

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

	df, err := os.Open(driverfile)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	defer df.Close()

	tree, err := toml.LoadReader(df)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	if err := config.ValidateDrivers(tree); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	drivers := driver.NewRegistry()

	for _, name := range tree.Keys() {
		drivers.Register(name, driverInits[name])
	}

	placer := block.NewFilesystem(objectsdir)

	if err := placer.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], errors.Cause(err))
		os.Exit(1)
	}

	collector := block.NewFilesystem(artifactsdir)

	if err := collector.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], errors.Cause(err))
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
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
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

	if err := r.Run(ctx, d); err != nil {
		os.Exit(1)
	}
}
