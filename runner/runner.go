package runner

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
)

var (
	errStageNotFound = errors.New("stage could not be found")
	errRunFailed     = errors.New("run failed")
)

type Runner struct {
	order     []string
	lastJob   *Job
	signals   chan os.Signal
	Out       io.Writer
	Objects   []config.Passthrough
	Stages    map[string]*Stage
	Collector Collector
}

func NewRunner(w io.Writer, objects []config.Passthrough, c Collector, signals chan os.Signal) *Runner {
	return &Runner{
		signals:   signals,
		Out:       w,
		Objects:   objects,
		Stages:    make(map[string]*Stage),
		Collector: c,
	}
}

func runJobs(jobs JobStore, d Driver, c Collector) chan *Job {
	wg := &sync.WaitGroup{}
	done := make(chan *Job)

	for _, j := range jobs {
		wg.Add(1)

		go func(j *Job) {
			defer wg.Done()

			d.Execute(j, c)

			done <- j

			after := runJobs(j.After, d, c)

			for a := range after {
				done <- a
			}
		}(j)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	return done
}

func (r *Runner) Add(stages ...*Stage) {
	for _, s := range stages {
		_, ok := r.Stages[s.Name]

		if !ok {
			r.Stages[s.Name] = s
			r.order = append(r.order, s.Name)
		}
	}
}

func (r *Runner) Remove(stages ...string) {
	for _, s := range stages {
		if _, ok := r.Stages[s]; !ok {
			continue
		}

		delete(r.Stages, s)

		i := 0

		for j, removed := range r.order {
			if removed == s {
				i = j
				break
			}
		}

		r.order = append(r.order[:i], r.order[i + 1:]...)
	}
}

func (r *Runner) Run(d Driver) error {
	defer d.Destroy()

	if err := d.Create(r.Out, r.Objects); err != nil {
		fmt.Fprintf(r.Out, "%s\n", errors.Cause(err))
		r.printLastJobStatus()

		return errRunFailed
	}

	for _, name := range r.order {
		if err := r.realRunStage(name, d); err != nil {
			if err == errStageNotFound {
				return err
			}

			break
		}
	}

	r.printLastJobStatus()

	if r.lastJob != nil && !r.lastJob.Success {
		return errRunFailed
	}

	return nil
}

func (r *Runner) RunStage(name string, d Driver) error {
	defer d.Destroy()

	if err := d.Create(r.Out, r.Objects); err != nil {
		fmt.Fprintf(r.Out, "%s\n", errors.Cause(err))
		r.printLastJobStatus()

		return errRunFailed
	}

	if err := r.realRunStage(name, d); err != nil {
		if err == errStageNotFound {
			return err
		}
	}

	r.printLastJobStatus()

	if !r.lastJob.Success {
		return errRunFailed
	}

	return nil
}

func (r Runner) printLastJobStatus() {
	if r.lastJob == nil {
		fmt.Fprintf(r.Out, "Done. No jobs run.\n")
		return
	}

	if !r.lastJob.Success {
		for _, err := range r.lastJob.Errors {
			fmt.Fprintf(r.Out, "error: %s\n", err)
		}

		if len(r.lastJob.Errors) > 0 {
			fmt.Fprintf(r.Out, "\n")
		}

		fmt.Fprintf(r.Out, "Done. Run failed.\n")
		return
	}

	fmt.Fprintf(r.Out, "Done. Run passed.\n")
}

func (r *Runner) realRunStage(name string, d Driver) error {
	stage, ok := r.Stages[name]

	if !ok {
		return errStageNotFound
	}

	if len(stage.Jobs) == 0 {
		return nil
	}

	jobs := runJobs(stage.Jobs, d, r.Collector)

	for jobs != nil {
		select {
			case sig := <-r.signals:
				if sig == os.Kill || sig == os.Interrupt {
					fmt.Fprintf(r.Out, "%s\n", sig)
					return errors.New("interrupt")
				}
			case j, ok := <-jobs:
				if !ok {
					jobs = nil
				} else {
					r.lastJob = j

					if !j.Success {
						fmt.Fprintf(r.Out, "\n")
						return errors.New("failed to run job: " + j.Name)
					}
				}
		}
	}

	fmt.Fprintf(r.Out, "\n")

	return nil
}
