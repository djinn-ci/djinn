package runner

import (
	"fmt"
	"io"
	"sync"

	"github.com/andrewpillar/thrall/errors"
)

var (
	errStageNotFound = errors.New("stage could not be found")
	errRunFailed     = errors.New("run failed")
)

type Runner struct {
	order   []string
	lastJob *Job
	Out     io.Writer
	Stages  map[string]*Stage
}

func NewRunner(w io.Writer) *Runner {
	return &Runner{
		Out:    w,
		Stages: make(map[string]*Stage),
	}
}

func runJobs(jobs JobStore, d Driver) chan *Job {
	wg := &sync.WaitGroup{}
	done := make(chan *Job)

	for _, j := range jobs {
		wg.Add(1)

		go func(j *Job) {
			defer wg.Done()

			d.Execute(j)

			done <- j

			after := runJobs(j.After, d)

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

func (r *Runner) Run(d Driver) error {
	if err := d.Create(r.Out); err != nil {
		return errors.Err(err)
	}

	defer d.Destroy()

	for _, name := range r.order {
		if err := r.RunStage(name, d); err != nil {
			if err == errStageNotFound {
				return err
			}

			break
		}
	}

	fmt.Fprintf(r.Out, "%s\n", r.lastJob.Status())

	if !r.lastJob.Success {
		return errRunFailed
	}

	return nil
}

func (r *Runner) RunStage(name string, d Driver) error {
	stage, ok := r.Stages[name]

	if !ok {
		return errStageNotFound
	}

	jobs := runJobs(stage.Jobs, d)

	for j := range jobs {
		io.Copy(r.Out, j.Buffer)

		r.lastJob = j

		if !j.Success {
			fmt.Fprintf(r.Out, "\n")
			return errors.New("failed to run job: " + j.Name)
		}
	}

	fmt.Fprintf(r.Out, "\n")

	return nil
}
