// Package runner providers various structs and interfaces for Running
// arbitrary Jobs in a Driver.
package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/andrewpillar/djinn/errors"
)

var (
	errStageNotFound = errors.New("stage could not be found")
	errTimedOut      = errors.New("timed out")
	errRunFailed     = errors.New("run failed")

	createTimeout   = time.Duration(time.Minute * 5)
	contextStatuses = map[error]Status{
		context.Canceled:         Killed,
		context.DeadlineExceeded: TimedOut,
	}
)

type jobHandler func(Job)

// Placer is the interface that defines how files from a host will be placed
// into a Driver during execution of the Runner. Files that are placed into a
// Driver are known as Objects to the Runner.
type Placer interface {
	// Place will take the object of the given name and write its contents to
	// the given io.Writer. The number of bytes written from the given
	// io.Writer are returned, along with any errors that occur.
	Place(string, io.Writer) (int64, error)

	// Stat will return the os.FileInfo of the object of the given name.
	Stat(string) (os.FileInfo, error)
}

// Collector is the interface that defines how files from the Driver will be
// collected from the Driver during execution of the Runner. Files that are
// collected from a Driver are known as Artifacts to the Runner.
type Collector interface {
	// Collect will read from the given io.Reader and store what was read as an
	// artifact under the given name. The number of bytes read from the given
	// io.Reader are returned, along with any errors that occur.
	Collect(string, io.Reader) (int64, error)
}

// Driver is the interface that defines how a Job should be executed for the
// Runner.
type Driver interface {
	// Each driver should implement the io.Writer interface, so that the driver
	// can write the output of what it's doing to the underlying io.Writer
	// implementation, for example os.Stdout.
	io.Writer

	// Create should create the Driver, and prepare it so it will be ready for
	// Jobs to be executed on it. It takes a context that will be used to cancel
	// out of the creation of the Driver quickly. The env slice of strings are
	// the environment variables that will be set on the driver, the strings in
	// the slice are formatted as key=value. The given Placer will be used to
	// place the given objects in the driver.
	Create(context.Context, []string, Passthrough, Placer) error

	// Execute should run the given Job on the Driver, and use the given
	// Collector, to collect any Artifacts for that Job. If the Job fails then
	// it should be marked as failed via the Failed method.
	Execute(*Job, Collector)

	// Destroy should render the driver unusable for executing Jobs, and clear
	// up any resources that may have been created via the Driver.
	Destroy()
}

// Runner is the struct for executing Jobs. Jobs are grouped together into
// stages. The order in which stages are added to the Runner is the order in
// which they will be executed when the Runner is run. The runner expects
// to have an underlying io.Writer, to which progress of each Stage and Job
// being executed will be written.
type Runner struct {
	io.Writer

	handleDriverCreate func()
	handleJobStart     jobHandler
	handleJobComplete  jobHandler

	// the order in which each stage is executed.
	order  []string
	stages map[string]*Stage

	// the last job that was successfully executed, used for reporting.
	lastJob Job

	// Env is a slice of environment variables to set during job exectuion. The
	// variables are expected to be formatted as key=value.
	Env []string

	// Objects are the files we want to place into the driver during Job
	// execution.
	Objects Passthrough

	// Placer is what to use for placing objects into the driver during Job
	// execution.
	Placer Placer

	// Collect is what to use for collecting artifacts from the driver during
	// Job execution.
	Collector Collector

	// Status is the status of the Runner was a run has been completed.
	Status Status
}

// Stage contains the jobs to run. The order in which the jobs are added to
// a Stage is the order in which they're executed.
type Stage struct {
	jobs jobStore

	// Name is the name of the Stage. Stage names are unqiue to a runner.
	Name string

	// CanFail denotes whether or not it is acceptable for a Stage to fail.
	// This is applied to each Job that is added to a Stage.
	CanFail bool
}

// HandleDriverCreate sets the given callback as the underlying handler for
// driver creation. This would typically be used for capturing timing
// information regarding driver creation, for example
func (r *Runner) HandleDriverCreate(f func()) { r.handleDriverCreate = f }

// HandleJobComplete sets the given callback as the underlying handler for Job
// completion. This will be passed the job that was just completed.
func (r *Runner) HandleJobComplete(f jobHandler) { r.handleJobComplete = f }

// HandleJobStart sets the given callback as the underlying handler for when a
// Job starts. This will be passed the Job that just started.
func (r *Runner) HandleJobStart(f jobHandler) { r.handleJobStart = f }

// Add adds the given stages to the Runner. The stages are stored in an
// underlying map where the key is the name of the Stage. A slice of the stage
// names is used to maintain the order in which stages are added.
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

// Remove removes the given Stages from the Runner based off the given names.
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
		r.order = append(r.order[:i], r.order[i+1:]...)
	}
}

// Run executes the Runner using the given Driver. The given context.Context is
// used to handle cancellation of the Runner, as well as cancellation of the
// underlying Driver during creation. If the Runner fails the errRunFailed will
// be returned.
func (r *Runner) Run(c context.Context, d Driver) error {
	defer d.Destroy()

	ct, cancel := context.WithTimeout(c, createTimeout)
	defer cancel()

	if r.handleDriverCreate != nil {
		r.handleDriverCreate()
	}

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
		if r.Status == TimedOut {
			return errTimedOut
		}
		return nil
	}
}

func (r *Runner) printLastJobStatus() {
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

	if stage.jobs.len() == 0 {
		return nil
	}

	for {
		j, ok := stage.jobs.next()

		if !ok {
			break
		}

		if len(j.Commands) > 0 {
			if r.handleJobStart != nil {
				r.handleJobStart(*j)
			}
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

// Stages returns the underlying map of the stages for the Runner.
func (r *Runner) Stages() map[string]*Stage { return r.stages }

// Add adds the given Jobs to the current Stage. The order in which the Jobs are
// added are the order in which they are executed.
func (s *Stage) Add(jobs ...*Job) {
	for _, j := range jobs {
		j.Stage = s.Name
		j.canFail = s.CanFail
		s.jobs.put(j)
	}
}

// Get returns the Job of the given name, and a boolean value denoting if the
// given Job exists in the current Stage.
func (s *Stage) Get(name string) (*Job, bool) { return s.jobs.get(name) }
