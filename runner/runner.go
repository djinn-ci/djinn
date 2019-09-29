package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/andrewpillar/thrall/errors"
)

var (
	errStageNotFound = errors.New("stage could not be found")
	errRunFailed     = errors.New("run failed")

	createTimeout = time.Duration(time.Minute * 5)

	contextStatuses = map[error]Status{
		context.Canceled:         Killed,
		context.DeadlineExceeded: TimedOut,
	}
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
	Status    Status
}

type Stage struct {
	jobs jobStore

	Name    string
	CanFail bool
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

func (r *Runner) Run(c context.Context, d Driver) error {
	defer d.Destroy()

	ct, cancel := context.WithTimeout(c, createTimeout)
	defer cancel()

	if err := d.Create(ct, r.Env, r.Objects, r.Placer); err != nil {
		cause := errors.Cause(err)

		fmt.Fprintf(d, "%s\n", cause.Error())
		r.printLastJobStatus()

		status, ok := contextStatuses[cause]

		if !ok {
			r.Status = Failed
		} else {
			r.Status = status
		}

		return errRunFailed
	}

	done := make(chan struct{})

	go func() {
		for _, name := range r.order {
			if err := r.realRunStage(name, d); err != nil {
				if err == errStageNotFound {
					done <- struct{}{}
					return
				}

				break
			}
		}

		done <- struct{}{}
	}()

	select {
	case <-c.Done():
		r.printLastJobStatus()

		err := c.Err()

		r.Status = contextStatuses[err]

		return err
	case <-done:
		if r.Status == Failed {
			return errRunFailed
		}

		return nil
	}
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

	for _, j := range stage.jobs {
		if len(j.Commands) > 0 {
			d.Execute(j, r.Collector)
		}

		if r.handleJobComplete != nil {
			r.handleJobComplete(*j)
		}

		r.lastJob = *j

		if j.Status >= r.Status {
			r.Status = j.Status
		}

		if r.Status == Failed {
			fmt.Fprintf(r.Writer, "\n")

			for _, err := range j.errs {
				fmt.Fprintf(r.Writer, "ERR: %s\n", err)
			}

			return errors.New("failed to run job: " + j.Name)
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
