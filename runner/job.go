package runner

import (
	"io"

	"github.com/andrewpillar/thrall/config"
)

type Job struct {
	Stage string
	Name  string

	Commands []string

	Errors []error

	Success bool
	CanFail bool
	DidFail bool

	Depends   []string
	Artifacts []config.Passthrough

	After  JobStore
	Writer io.Writer
}

type JobStore map[string]*Job

func NewJob(w io.Writer, name string, commands, depends []string, artifacts []config.Passthrough) *Job {
	j := &Job{
		Name:      name,
		Commands:  commands,
		Depends:   depends,
		Artifacts: artifacts,
		Errors:    make([]error, 0),
		After:     NewJobStore(),
		Writer:    w,
	}

	return j
}

func NewJobStore() JobStore {
	return JobStore(make(map[string]*Job))
}

// Mark a job as failed. The only errors that should be passed to this method should be errors
// pertaining to the functionality of the driver executing the job.
func (j *Job) Failed(err error) {
	if err != nil {
		j.Errors = append(j.Errors, err)
	}

	j.Success = j.CanFail
	j.DidFail = true
}

func (s JobStore) Get(name string) (*Job, bool) {
	j, ok := s[name]

	if ok {
		return j, ok
	}

	for _, j := range s {
		after, ok := j.After.Get(name)

		if ok {
			return after, ok
		}
	}

	return nil, false
}

func (s *JobStore) Put(j *Job) {
	if len(j.Depends) == 0 {
		(*s)[j.Name] = j
		return
	}

	for _, d := range j.Depends {
		before, ok := s.Get(d)

		if ok {
			before.After.Put(j)
			return
		}
	}

	(*s)[j.Name] = j
}
