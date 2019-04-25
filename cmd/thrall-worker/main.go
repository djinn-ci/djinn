package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/collector"
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/placer"
	"github.com/andrewpillar/thrall/runner"
	"github.com/andrewpillar/thrall/queue"

	"github.com/go-redis/redis"

	"github.com/lib/pq"
)

var (
	Builds map[int64]chan os.Signal

	Client  *redis.Client
	Drivers []string

	Placer    runner.Placer
	Collector runner.Collector

	SSHKey     string
	SSHTimeout int

	QemuDir    string
	QemuCPUs   int
	QemuMemory int
	QemuPort   int
	QemuUser   string
)

func runBuild(id int64, smanifest string) {
	log.Debug.Println("received task for build:", id)

	manifest, err := config.DecodeManifest(strings.NewReader(smanifest))

	if err != nil {
		log.Error.Println(errors.Err(err))
		return
	}

	b, err := model.FindBuild(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		return
	}

	if b.IsZero() {
		log.Error.Println("failed to find build:", id)
		return
	}

	if b.Manifest != smanifest {
		b.Status = model.Failed

		if err := b.Update(); err != nil {
			log.Error.Println(errors.Err(err))
			return
		}

		return
	}

	if err := b.LoadRelations(); err != nil {
		log.Error.Println(errors.Err(err))
		return
	}

	if err := model.LoadStageJobs(b.Stages); err != nil {
		log.Error.Println(errors.Err(err))
		return
	}

	b.Status = model.Running
	b.StartedAt = &pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := b.Update(); err != nil {
		log.Error.Println(errors.Err(err))
		return
	}

	buildOut := &bytes.Buffer{}
	driverOut := &bytes.Buffer{}

	Builds[id] = make(chan os.Signal)

	r := runner.NewRunner(
		buildOut,
		manifest.Env,
		manifest.Objects,
		placer.NewDatabase(Placer),
		collector.NewDatabase(Collector),
		Builds[id],
	)

	for _, s := range build.Stages {
		r.Add(runner.NewStage(s.Name, s.CanFail))
	}

	var d runner.Driver

	validDriver := true

	switch manifest.Driver.Type {
		case "docker":
			d = driver.NewDocker(manifest.Driver.Image, manifest.Driver.Workspace)
		case "qemu":
			d = &driver.Qemu{
				Writer: driverOut,
				SSH:    &driver.SSH{
					Writer:   ioutil.Discard,
					Address:  hostfwd,
					Username: QemuUser,
					KeyFile:  SSHKey,
					Timeout:  time.Duration(time.Second * time.Duration(SSHTimeout)),
				},
				Image:   manifest.Driver.Image,
				Arch:    manifest.Driver.Arch,
				HostFwd: hostfwd,
			}
		case "ssh":
			d = &driver.SSH{
				Wirter:   driverOut,
				Address:  manifest.Driver.Address,
				Username: manifest.Driver.Username,
				KeyFile:  SSHKey,
				Timeout:  time.Duration(time.Second * time.Duration(SSHTimeout)),
			}
		default:
			validDriver = false
	}

	if !validDriver {
		log.Error.Println("invalid driver:", manifest.Driver.Type)
		return
	}

	if err := r.Run(d); err != nil {
		b.Status = model.Failed
	} else {
		b.Status = model.Passed
	}
}

func mainCommand(c cli.Command) {
	f, err := os.Open(c.Flags.GetString("config"))

	if err != nil {
		log.Error.Fatalf("failed to open worker config: %s\n", err)
	}

	defer f.Close()

	cfg, err := config.DecodeWorker(f)

	if err != nil {
		log.Error.Fatalf("failed to decode worker config: %s\n", err)
	}

	SSHKey = cfg.SSH.Key
	SSHTimeout = cfg.SSH.Timeout

	QemuDir = cfg.Qemu.Dir
	QemuCPUs = cfg.Qemu.CPUs
	QemuMemory = cfg.Qemu.Memory
	QemuPort = cfg.Qemu.Port
	QemuUser = cfg.Qemu.User

	switch cfg.Objects.Type {
		case "filesystem":
			Placer = placer.NewFileSystem(cfg.Objects.Dir)
		default:
			log.Error.Fatalf("unknown object storage type:", cfg.Objects.Type)
	}

	switch cfg.Artifacts.Type {
		case "filesystem":
			Collector = collector.NewFileSystem(cfg.Artifacts.Dir)
		default:
			log.Error.Fatalf("unknown artifact type:", cfg.Artifacts.Type)
	}

	server, err := queue.New(queue.Builds, cfg.Redis.Addr, cfg.Redis.Password)

	if err != nil {
		log.Error.Fatalf("failed to create queue: %s\n", err)
	}

	Builds = make(map[int64]chan os.Signal)

	if err := server.RegisterTask("run_build", runBuild); err != nil {
		log.Error.Fatalf("failed to register task: %s\n", err)
	}

	worker := server.NewWorker("thrall_worker", cfg.Parallelism)

	if err := worker.Launch(); err != nil {
		log.Error.Fatalf("failed to launch worker: %s\n", err)
	}
}

func main() {
	c := cli.New()

	cmd := c.MainCommand(mainCommand)

	cmd.AddFlag(&cli.Flag{
		Name:     "config",
		Short:    "-c",
		Long:     "--config",
		Argument: true,
		Default:  "thrall-worker.toml",
	})

	if err := c.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}
}
