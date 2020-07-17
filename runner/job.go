package runner

import "io"

// Job is the struct that represents a Job to be executed in a Driver. Similar
// to the Runner it has an underlying io.Writer that will have progress of the
// Job execution written to it. Each Job will belong to a Stage, and will be
// executed in the order that Job was added to the Stage.
type Job struct {
	io.Writer

	errs    []error // errors that occurred during Job execution.
	canFail bool    // canFail denotes if the job can fail, this is set when a
	// Job is added to a Stage.

	// Stage is the name of the Stage to which the Job belongs.
	Stage string

	// Name is the name of the Job.
	Name string

	// Commands is the list of the commands that should be executed in the
	// Driver when the Job is executed.
	Commands []string

	// Artifacts is the Passthrough that denotes how Artifacts in the Driver
	// should map to the host.
	Artifacts Passthrough

	// Status is the Status of the Job once it has completed execution.
	Status Status
}

// jobStore is the struct that holds the jobs for a Stage. Jobs are stored in a
// map where the key is the Job's name. Order is maintained via a slice of the
// Job names. The field curr is an integer that points to the position in the
// order slice.
type jobStore struct {
	order []string
	curr  int
	jobs  map[string]*Job
}

func (j Job) isZero() bool {
	return j.Writer == nil &&
		len(j.errs) == 0 &&
		!j.canFail &&
		j.Stage == "" &&
		j.Name == "" &&
		len(j.Commands) == 0 &&
		j.Artifacts.Values == nil &&
		j.Status == Status(0)
}

// Mark a job as failed. The only errors that should be passed to this method
// should be errors pertaining to the functionality of the driver executing
// the job.
func (j *Job) Failed(err error) {
	if err != nil {
		j.errs = append(j.errs, err)
	}

	if j.canFail {
		j.Status = PassedWithFailures
	} else {
		j.Status = Failed
	}
}

func (j jobStore) len() int { return len(j.jobs) }

// next returns the next Job in the jobStore to be executed. This will
// increment the underlying curr field. If there is no Job to be executed
// then a false value is returned for the bool value.
func (s *jobStore) next() (*Job, bool) {
	if s.curr >= len(s.order) {
		return nil, false
	}

	j, ok := s.jobs[s.order[s.curr]]
	s.curr++
	return j, ok
}

func (s *jobStore) get(name string) (*Job, bool) {
	j, ok := s.jobs[name]
	return j, ok
}

func (s *jobStore) put(j *Job) {
	if s.order == nil {
		s.order = make([]string, 0)
	}
	if s.jobs == nil {
		s.jobs = make(map[string]*Job)
	}

	s.order = append(s.order, j.Name)
	s.jobs[j.Name] = j
}
