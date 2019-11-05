package runner

import "io"

type Job struct {
	io.Writer

	errs    []error
	canFail bool
	after   jobStore

	Stage     string
	Name      string
	Commands  []string
	Artifacts Passthrough
	Status    Status
}

type jobStore map[string]*Job

func (j Job) isZero() bool {
	return j.Writer == nil &&
           len(j.errs) == 0 &&
           !j.canFail &&
           j.after == nil &&
           j.Stage == "" &&
           j.Name == "" &&
           len(j.Commands) == 0 &&
           j.Artifacts.vals == nil &&
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

func (s jobStore) Get(name string) (*Job, bool) {
	if j, ok := s[name]; ok {
		return j, ok
	}

	for _, j := range s {
		after, ok := j.after.Get(name)

		if ok {
			return after, ok
		}
	}

	return nil, false
}

func (s *jobStore) Put(j *Job) {
	if (*s) == nil {
		(*s) = make(map[string]*Job)
	}

	(*s)[j.Name] = j
}
