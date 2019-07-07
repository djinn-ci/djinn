package runner

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/andrewpillar/thrall/errors"
)

var (
	errStageNotFound = errors.New("stage could not be found")
	errRunFailed     = errors.New("run failed")
)

type jobHandler func(j Job)

// Placer allows for placing files into the given io.Writer.
type Placer interface {
	Place(name string, w io.Writer) (int64, error)

	Stat(name string) (os.FileInfo, error)
}

// Collector allows for collecting files from the given io.Reader.
type Collector interface {
	Collect(name string, r io.Reader) (int64, error)
}

type Runner struct {
	io.Writer

	handleJobStart    jobHandler
	handleJobComplete jobHandler

	order   []string
	stages  map[string]*Stage
	lastJob Job

	Env       []string
	Objects   Passthrough
	Placer    Placer
	Collector Collector
	Signals   chan os.Signal
	Status    Status
}

type Stage struct {
	jobs jobStore

	Name    string
	CanFail bool
}

func runJobs(jobs jobStore, d Driver, c Collector, fn jobHandler) chan Job {
	wg := &sync.WaitGroup{}
	done := make(chan Job)

	for _, j := range jobs {
		wg.Add(1)

		go func(j *Job) {
			defer wg.Done()

			if fn != nil {
				fn(*j)
			}

			d.Execute(j, c)

			done <- *j

			after := runJobs(j.after, d, c, fn)

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
	if r.stages == nil {
		r.stages = make(map[string]*Stage)
	}

	for _, s := range stages {
		_, ok := r.stages[s.Name]

		if !ok {
			r.stages[s.Name] = s
			r.order = append(r.order, s.Name)
		}
	}
}

func (r *Runner) Remove(stages ...string) {
	for _, s := range stages {
		if _, ok := r.stages[s]; !ok {
			continue
		}

		delete(r.stages, s)

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

	if err := d.Create(r.Env, r.Objects, r.Placer); err != nil {
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

func (r *Runner) RunWithTimeout(d Driver, duration time.Duration) error {
	defer d.Destroy()

	if err := d.Create(r.Env, r.Objects, r.Placer); err != nil {
		fmt.Fprintf(r.Writer, "%s\n", errors.Cause(err))
		r.printLastJobStatus()

		r.Status = Failed

		return errRunFailed
	}

	go func() {
		<-time.After(duration)
		r.Signals <- TimedOut
	}()

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
	if r.lastJob.isZero() {
		fmt.Fprintf(r.Writer, "Done. No jobs run.\n")
		return
	}

	for _, err := range r.lastJob.errs {
		fmt.Fprintf(r.Writer, "error: %s\n", err)
	}

	if len(r.lastJob.errs) > 0 {
		fmt.Fprintf(r.Writer, "\n")
	}

	fmt.Fprintf(r.Writer, "Done. Run %s\n", r.Status)
}

func (r *Runner) realRunStage(name string, d Driver) error {
	stage, ok := r.stages[name]

	if !ok {
		return errStageNotFound
	}

	if len(stage.jobs) == 0 {
		return nil
	}

	jobs := runJobs(stage.jobs, d, r.Collector, r.handleJobStart)

	for jobs != nil {
		select {
			case sig := <-r.Signals:
				if sig == os.Kill || sig == os.Interrupt || sig == TimedOut {
					if sig == TimedOut {
						r.Status = TimedOut
					} else {
						r.Status = Killed
					}

					return errors.New(sig.String())
				}
			case j, ok := <-jobs:
				if !ok {
					jobs = nil
				} else {
					if r.handleJobComplete != nil {
						r.handleJobComplete(j)
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

func (r Runner) Stages() map[string]*Stage {
	return r.stages
}

func (s *Stage) Add(jobs ...*Job) {
	for _, j := range jobs {
		j.Stage = s.Name
		j.canFail = s.CanFail

		s.jobs.Put(j)
	}
}
