package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/andrewpillar/thrall/collector"
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/placer"
	"github.com/andrewpillar/thrall/runner"

	"github.com/go-redis/redis"

	"github.com/lib/pq"
)

type Worker struct {
	Client    *redis.Client
	Drivers   []string
	Placer    *placer.Database
	Collector *collector.Database
	Signals   chan os.Signal

}

func (w Worker) RunBuild(id int64, smanifest string) {
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

	b.Status = model.Running
	b.StartedAt = &pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := b.Update(); err != nil {
		log.Error.Println(errors.Err(err))
		return
	}

	w.Placer.Build = b
	w.Collector.Build = b

	buildOut := &bytes.Buffer{}

	r := runner.NewRunner(
		buildOut,
		manifest.Env,
		manifest.Objects,
		w.Placer,
		w.Collector,
		w.Signals,
	)

	validDriver := true

	var d runner.Driver

	switch manifest.Driver.Type {
		case "docker":
			d = driver.NewDocker(manifest.Driver.Image, manifest.Driver.Workspace)
			break
		case "qemu":
			d = &driver.QEMU{
				Writer:
				SSH:     &driver.SSH{
					Writer:   ioutil.Discard,
					Address:  ,
					Username: ,
					KeyFile:  ,
					Timeout:  ,
				}
				Image:   manifest.Driver.Image,
				Arch:    manifest.Driver.Arch,
				HostFwd: hostfwd,
			}
			break
		case "ssh":
			d = &driver.SSH{
				Writer:   ,
				Address:  manifest.Driver.Address,
				Username: manifest.Driver.Username,
				KeyFile:  ,
				Timeout:  ,
			}
			break
		default:
			validDriver = false
	}

	if err := r.Run(d); err != nil {
		b.Status = model.Failed
	} else {
		b.Status = model.Passed
	}

	wg := &sync.WaitGroup{}

	for j := range r.Jobs {
		wg.Add(1)

		go func(j runner.Job) {
			defer wg.Done()

			mj, err := b.FindJobByName(j.Name)

			if err != nil {
				log.Error.Println(errors.Err(err))
				return
			}

			if j.Success && !j.DidFail {
				mj.Status = model.Passed
			} else if j.Success && j.DidFail {
				mj.Status = model.PassedWithFailures
			} else {
				mj.Status = model.Failed
			}

			if err := mj.Update(); err != nil {
				log.Error.Println(errors.Err(err))
			}
		}(j)
	}

	wg.Wait()

	b.Output = buildOut.String()
	b.FinishedAt = &pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := b.Update(); err != nil {
		log.Error.Println(errors.Err(err))
	}
}
