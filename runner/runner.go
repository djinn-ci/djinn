package runner

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"djinn-ci.com/errors"

	"github.com/andrewpillar/fs"
)

type Status uint8

//go:generate stringer -type Status -linecomment
const (
	Queued             Status = iota // queued
	Running                          // running
	Passed                           // passed
	PassedWithFailures               // passed_with_failures
	Failed                           // failed
	Killed                           // killed
	TimedOut                         // timed_out
)

func (s Status) MarshalJSON() ([]byte, error) { return json.Marshal(s.String()) }

var statusMap = map[string]Status{
	"queued":               Queued,
	"running":              Running,
	"passed":               Passed,
	"passed_with_failures": PassedWithFailures,
	"failed":               Failed,
	"killed":               Killed,
	"timed_out":            TimedOut,
}

func (s *Status) Scan(val any) error {
	v, err := driver.String.ConvertValue(val)

	if err != nil {
		return errors.Err(err)
	}

	str, ok := v.(string)

	if !ok {
		return fmt.Errorf("runner: cannot type assert %T to %T", val, str)
	}

	if str == "" {
		return nil
	}

	if err := s.UnmarshalText([]byte(str)); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Status) UnmarshalText(b []byte) error {
	var ok bool

	str := string(b)
	(*s), ok = statusMap[str]

	if !ok {
		return errors.New("runner: unknown status " + str)
	}
	return nil
}

func (s Status) Value() (driver.Value, error) { return driver.Value(s.String()), nil }

type Passthrough map[string]string

func (p Passthrough) MarshalYAML() (any, error) {
	if p == nil {
		return nil, nil
	}

	arr := make([]string, 0, len(p))

	for k, v := range p {
		arr = append(arr, k+" => "+v)
	}

	sort.Strings(arr)

	return arr, nil
}

func (p *Passthrough) UnmarshalYAML(unmarshal func(any) error) error {
	if (*p) == nil {
		(*p) = make(Passthrough)
	}

	arr := make([]string, 0)

	if err := unmarshal(&arr); err != nil {
		return err
	}

	for _, s := range arr {
		parts := strings.SplitN(s, "=>", 2)

		key := strings.TrimSpace(parts[0])
		val := filepath.Base(key)

		if len(parts) > 1 {
			val = strings.TrimSpace(parts[1])
		}
		(*p)[key] = val
	}
	return nil
}

type Job struct {
	io.Writer

	errs    []error
	canFail bool
	status  Status
	stage   string

	Name      string
	Commands  []string
	Artifacts Passthrough
}

func (j *Job) FullName() string {
	if j.stage == "" {
		return j.Name
	}
	return j.stage + "/" + j.Name
}

func (j *Job) Status() Status { return j.status }

func (j *Job) Failed(err error) {
	if err != nil {
		if !errors.Is(err, ErrFailed) {
			j.errs = append(j.errs, &Error{Stage: j.stage, Job: j.Name, Err: err})
		}
	}

	j.status = Failed

	if j.canFail {
		j.status = PassedWithFailures
	}
}

type orderedMap[T any] struct {
	order []string
	curr  int
	m     map[string]T
}

func (m *orderedMap[T]) len() int {
	if m == nil {
		return 0
	}
	return len(m.m)
}

func (m *orderedMap[T]) next() (T, bool) {
	var zero T

	if m.curr >= m.len() {
		return zero, false
	}

	it, ok := m.m[m.order[m.curr]]
	m.curr++

	return it, ok
}

func (m *orderedMap[T]) get(name string) (T, bool) {
	var zero T

	if m.m == nil {
		return zero, false
	}

	it, ok := m.m[name]

	return it, ok
}

func (m *orderedMap[T]) set(name string, t T) {
	if m.order == nil {
		m.order = make([]string, 0)
	}
	if m.m == nil {
		m.m = make(map[string]T)
	}

	if _, ok := m.m[name]; !ok {
		m.order = append(m.order, name)
	}
	m.m[name] = t
}

func (m *orderedMap[T]) delete(name string) {
	if m.m == nil {
		return
	}

	if _, ok := m.m[name]; !ok {
		return
	}

	delete(m.m, name)

	i := 0

	for j, s := range m.order {
		if name == s {
			i = j
			break
		}
	}
	m.order = append(m.order[:i], m.order[i+1:]...)
}

type Stage struct {
	jobs *orderedMap[*Job]

	Name    string
	CanFail bool
}

func (s *Stage) Add(jobs ...*Job) {
	if s.jobs == nil {
		s.jobs = &orderedMap[*Job]{}
	}

	for _, j := range jobs {
		j.stage = s.Name
		j.canFail = s.CanFail

		if j.Name == "" {
			j.Name = fmt.Sprintf("%s.%d", s.Name, s.jobs.len()+1)
		}
		s.jobs.set(j.Name, j)
	}
}

func (s *Stage) Get(name string) (*Job, bool) { return s.jobs.get(name) }

type Error struct {
	Stage string
	Job   string
	Err   error
}

func (e *Error) Error() string {
	return "job " + e.Job + " in stage " + e.Stage + " failed: " + e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

type Driver interface {
	io.Writer

	Create(ctx context.Context, env []string, pt Passthrough, fs fs.FS) error

	Execute(j *Job, fs fs.FS) error

	Destroy()
}

type JobHandlerFunc func(j *Job)

type Runner struct {
	io.Writer

	status Status
	stages *orderedMap[*Stage]
	job    *Job

	handleJobStart    JobHandlerFunc
	handleJobComplete JobHandlerFunc

	Env         []string
	Passthrough Passthrough

	Objects   fs.FS
	Artifacts fs.FS
}

func Default(pt Passthrough) *Runner {
	return &Runner{
		Writer:      os.Stdout,
		Env:         os.Environ(),
		Passthrough: pt,
		Objects:     fs.New("."),
		Artifacts:   fs.New("."),
	}
}

func (r *Runner) HandleJobStart(fn JobHandlerFunc)    { r.handleJobStart = fn }
func (r *Runner) HandleJobComplete(fn JobHandlerFunc) { r.handleJobComplete = fn }

func (r *Runner) Stages() []*Stage {
	if r.stages == nil {
		return nil
	}

	stages := make([]*Stage, 0, r.stages.len())

	for _, name := range r.stages.order {
		stages = append(stages, r.stages.m[name])
	}
	return stages
}

func (r *Runner) Add(stages ...*Stage) {
	if r.stages == nil {
		r.stages = &orderedMap[*Stage]{}
	}

	for _, st := range stages {
		r.stages.set(st.Name, st)
	}
}

func (r *Runner) Remove(names ...string) {
	for _, name := range names {
		r.stages.delete(name)
	}
}

func (r *Runner) printLastJobStatus() {
	if r.job == nil {
		fmt.Fprintf(r.Writer, "Done. No jobs runs.\n")
		return
	}

	for _, err := range r.job.errs {
		fmt.Fprintf(r.Writer, "error: %s\n", err)
	}

	if len(r.job.errs) > 0 {
		fmt.Fprintf(r.Writer, "\n")
	}
	fmt.Fprintf(r.Writer, "Done. Run %s\n", r.status)
}

func (r *Runner) runStage(st *Stage, d Driver) error {
	if st.jobs.len() == 0 {
		return nil
	}

	for {
		j, ok := st.jobs.next()

		if !ok {
			break
		}

		r.job = j

		j.status = Running

		if len(j.Commands) > 0 {
			if r.handleJobStart != nil {
				r.handleJobStart(j)
			}

			if err := d.Execute(j, r.Artifacts); err != nil {
				j.Failed(err)
			}
		}

		if j.status != Failed && j.status != PassedWithFailures {
			j.status = Passed
		}

		if r.handleJobComplete != nil {
			r.handleJobComplete(j)
		}

		if j.status >= r.status {
			r.status = j.status
		}

		if r.status == Failed {
			fmt.Fprintf(r.Writer, "\n")

			for _, err := range j.errs {
				fmt.Fprintf(r.Writer, "error: %s\n", err)
			}
		}
	}
	return nil
}

var (
	ErrFailed   = errors.New("runner: failed")
	ErrTimedOut = errors.New("runner: timed out")
)

func (r *Runner) updateUnfinishedJobs() {
	for _, st := range r.stages.m {
		if st.jobs == nil {
			continue
		}

		for _, j := range st.jobs.m {
			if j.status == Queued {
				j.status = r.status

				if r.handleJobComplete != nil {
					r.handleJobComplete(j)
				}
			}
		}
	}
}

func (r *Runner) findCreateDriverJob() *Job {
	for _, st := range r.stages.m {
		j, ok := st.jobs.get("create-driver")

		if ok {
			return j
		}
	}

	return &Job{
		Name: "create-driver",
	}
}

func (r *Runner) Run(ctx context.Context, d Driver) error {
	r.status = Running
	r.Objects = fs.ReadOnly(r.Objects)
	r.Artifacts = fs.WriteOnly(r.Artifacts)

	defer d.Destroy()

	ctx, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()

	statustab := map[error]Status{
		context.Canceled:         Killed,
		context.DeadlineExceeded: TimedOut,
	}

	if r.handleJobStart != nil {
		r.handleJobStart(r.findCreateDriverJob())
	}

	if err := d.Create(ctx, r.Env, r.Passthrough, r.Objects); err != nil {
		cause := errors.Cause(err)

		fmt.Fprintf(d, "%s\n", cause.Error())

		status, ok := statustab[cause]

		if !ok {
			status = Failed
		}

		r.status = status
		r.updateUnfinishedJobs()

		return ErrFailed
	}

	if r.handleJobComplete != nil {
		r.handleJobComplete(r.findCreateDriverJob())
	}

	done := make(chan struct{})

	go func() {
		for {
			st, ok := r.stages.next()

			if !ok {
				break
			}

			if err := r.runStage(st, d); err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		err := ctx.Err()

		r.status = statustab[err]
		r.updateUnfinishedJobs()

		return err
	case <-done:
		if r.status == Failed {
			return ErrFailed
		}
		if r.status == TimedOut {
			return ErrTimedOut
		}
		return nil
	}
}

func (r *Runner) Status() Status { return r.status }
