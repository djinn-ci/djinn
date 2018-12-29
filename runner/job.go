package runner

import (
	"bytes"
	"fmt"
	"io"
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
	Artifacts []string

	After   JobStore
	Buffer  io.ReadWriter
}

type JobStore map[string]*Job

func NewJob(name string, commands, depends, artifacts []string) *Job {
	j := &Job{
		Name:      name,
		Commands:  commands,
		Errors:    make([]error, 0),
		Depends:   depends,
		Artifacts: artifacts,
		After:     NewJobStore(),
		Buffer:    &bytes.Buffer{},
	}

	return j
}

func NewJobStore() JobStore {
	return JobStore(make(map[string]*Job))
}

func (j *Job) Failed() {
	j.Success = j.CanFail
	j.DidFail = true
}

func (j Job) Status() string {
	buf := &bytes.Buffer{}

	if !j.Success {
		if len(j.Errors) > 0 {
			fmt.Fprintf(buf, "\n")
		}

		for _, err := range j.Errors {
			fmt.Fprintf(buf, "%s\n", err)
		}

		fmt.Fprintf(buf, "\nDone. Run failed.\n")

		return buf.String()
	}

	return "\nDone. Run passed.\n"
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
