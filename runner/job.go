package runner

import "io"

type Job struct {
	io.Writer

	errs    []error
	canFail bool

	Stage     string
	Name      string
	Commands  []string
	Artifacts Passthrough
	Status    Status
}

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
