package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"djinn-ci.com/config"
	"djinn-ci.com/manifest"
	"djinn-ci.com/runner"
	"djinn-ci.com/version"

	"github.com/andrewpillar/fs"
)

func exiterr(err error) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
	os.Exit(1)
}

const setupStage = "setup"

func main() {
	var (
		showversion  bool
		artifactsdir string
		objectsdir   string
		manif        string
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
	flag.StringVar(&manif, "manifest", ".djinn.yml", "the manifest file to use")
	flag.StringVar(&driverfile, "driver", filepath.Join(cfgdir, "djinn", "driver.conf"), "the driver config to use")
	flag.StringVar(&stage, "stage", "", "the stage to execute")
	flag.Parse(os.Args[1:])

	if showversion {
		fmt.Printf("%s %s %s/%s\n", os.Args[0], version.Build, runtime.GOOS, runtime.GOARCH)
		return
	}

	f, err := os.Open(manif)

	if err != nil {
		exiterr(err)
	}

	defer f.Close()

	m, err := manifest.Decode(f)

	if err != nil {
		exiterr(err)
	}

	if err := m.Validate(); err != nil {
		exiterr(err)
	}

	typ := m.Driver["type"]

	// Force to x86_64 for now since this is all we support for the QEMU driver.
	// Down the line we perhaps may want to use the host arch as a default.
	if typ == "qemu" {
		m.Driver["arch"] = "x86_64"
	}

	f2, err := os.Open(driverfile)

	if err != nil {
		exiterr(err)
	}

	defer f2.Close()

	driverInit, driverCfg, err := config.DecodeDriver(typ, f2.Name(), f2)

	if err != nil {
		exiterr(err)
	}

	r := runner.Default(m.Objects)
	r.Env = m.Env
	r.Artifacts = fs.New(artifactsdir)
	r.Objects = fs.New(objectsdir)

	setup := runner.Stage{
		Name: fmt.Sprintf("%s - %v", setupStage, time.Now().Unix()),
	}

	for i, src := range m.Sources {
		name := fmt.Sprintf("clone.%d", i+1)

		commands := []string{"git clone " + src.URL + " " + src.Dir}

		if src.Ref != "" {
			commands = append(commands, "cd "+src.Dir, "git checkout -q "+src.Ref)
		}

		if src.Dir != "" {
			commands = append([]string{"mkdir -p " + src.Dir}, commands...)
		}

		setup.Add(&runner.Job{
			Writer:   os.Stdout,
			Name:     name,
			Commands: commands,
		})
	}

	r.Add(&setup)

	failtab := make(map[string]struct{}, len(m.AllowFailures))

	for _, stage := range m.AllowFailures {
		failtab[stage] = struct{}{}
	}

	stagetab := make(map[string]*runner.Stage)

	for _, name := range m.Stages {
		_, ok := failtab[name]

		stagetab[name] = &runner.Stage{
			Name:    name,
			CanFail: ok,
		}
	}

	if stage != "" && stage != setupStage {
		for _, name := range m.Stages {
			if name != stage {
				delete(stagetab, name)
			}
		}
	}

	for _, j := range m.Jobs {
		stage, ok := stagetab[j.Stage]

		if !ok {
			continue
		}

		stage.Add(&runner.Job{
			Writer:    os.Stdout,
			Name:      j.Name,
			Commands:  j.Commands,
			Artifacts: j.Artifacts,
		})
	}

	for _, name := range m.Stages {
		r.Add(stagetab[name])
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)

	go func() {
		<-ch
		cancel()
	}()

	if err := r.Run(ctx, driverInit(os.Stdout, driverCfg.Merge(m.Driver))); err != nil {
		exiterr(err)
	}
}
