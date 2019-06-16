package runner

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/andrewpillar/thrall/errors"
)

var (
	errStageNotFound = errors.New("stage could not be found")
	errRunFailed     = errors.New("run failed")
)

type jobHandler func(j Job)

type Placer interface {
	Place(name string, w io.Writer) (int64, error)
}

type Collector interface {
	Collect(name string, r io.Reader) (int64, error)
}

type Runner struct {
	io.Writer

	handleJobStart    jobHandler
	handleJobComplete jobHandler

	order     []string
	lastJob   *Job
	sigs      chan os.Signal
	env       []string
	objs      Passthrough
	placer    Placer
	collector Collector

	Status Status
	Stages map[string]*Stage
}

func NewRunner(
	w    io.Writer,
	env  []string,
	objs Passthrough,
	p    Placer,
	c    Collector,
	sigs chan os.Signal,
) *Runner {
	return &Runner{
		Writer:    w,
		sigs:      sigs,
		env:       env,
		objs:      objs,
		placer:    p,
		collector: c,
		Stages:    make(map[string]*Stage),
	}
}

func runJobs(jobs JobStore, d Driver, c Collector, fn jobHandler) chan *Job {
	wg := &sync.WaitGroup{}
	done := make(chan *Job)

	for _, j := range jobs {
		wg.Add(1)

		go func(j *Job) {
			defer wg.Done()

			if fn != nil {
				fn(*j)
			}

			d.Execute(j, c)

			done <- j

			after := runJobs(j.After, d, c, fn)

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

func (r *Runner) HandleJobComplete(f jobHandler) {
	r.handleJobComplete = f
}

func (r *Runner) HandleJobStart(f jobHandler) {
	r.handleJobStart = f
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
	r.Status = Running

	defer d.Destroy()

	if err := d.Create(r.env, r.objs, r.placer); err != nil {
		fmt.Fprintf(r.Writer, "%s\n", errors.Cause(err))
		r.printLastJobStatus()

		r.Status = Failed

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

	if r.Status == Failed {
		return errRunFailed
	}

	return nil
}

func (r Runner) printLastJobStatus() {
	if r.lastJob == nil {
		fmt.Fprintf(r.Writer, "Done. No jobs run.\n")
		return
	}

	for _, err := range r.lastJob.Errors {
		fmt.Fprintf(r.Writer, "error: %s\n", err)
	}

	if len(r.lastJob.Errors) > 0 {
		fmt.Fprintf(r.Writer, "\n")
	}

	fmt.Fprintf(r.Writer, "Done. Run %s\n", r.Status)
}

func (r *Runner) realRunStage(name string, d Driver) error {
	stage, ok := r.Stages[name]

	if !ok {
		return errStageNotFound
	}

	if len(stage.Jobs) == 0 {
		return nil
	}

	jobs := runJobs(stage.Jobs, d, r.collector, r.handleJobStart)

	for jobs != nil {
		select {
			case sig := <-r.sigs:
				if sig == os.Kill || sig == os.Interrupt {
					fmt.Fprintf(r.Writer, "%s\n", sig)
					return errors.New("interrupt")
				}
			case j, ok := <-jobs:
				if !ok {
					jobs = nil
				} else {
					if r.handleJobComplete != nil {
						r.handleJobComplete(*j)
					}

					r.lastJob = j

					if j.Status >= r.Status {
						r.Status = j.Status
					}

					if r.Status == Failed {
						fmt.Fprintf(r.Writer, "\n")
						return errors.New("failed to run job: " + j.Name)
					}
				}
		}
	}

	fmt.Fprintf(r.Writer, "\n")

	return nil
}
