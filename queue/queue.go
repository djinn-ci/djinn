// Package queue provides the Queue interface for the queueing and processing
// of background jobs.
package queue

import (
	"bytes"
	"context"
	"encoding/gob"
	"runtime/debug"
	"sync"

	"djinn-ci.com/errors"
	"djinn-ci.com/log"

	"github.com/mcmathja/curlyq"
)

var queueName = "jobs"

// InitFunc is a callback for initializing a Job for it to be performed. This
// would typically be used for setting up dependencies for a Job that could not
// otherwise be reliably stored on the Queue, such as database connections.
type InitFunc func(Job)

type Job interface {
	// Name returns the name of the Job being performed. This is used to lookup
	// the resptive InitFunc for the Job, if any.
	Name() string

	// Perform performs the Job on the Queue. This should return any errors
	// if the Job failed in a fatal away that cannot be recovered from.
	Perform() error
}

// InitRegistry provides a thread-safe mechanism of registering an InitFunc
// against a Job name.
type InitRegistry struct {
	mu  *sync.RWMutex
	fns map[string]InitFunc
}

type Set struct {
	mu     *sync.RWMutex
	queues map[string]Queue
}

type Queue interface {
	// InitFunc registers the given init callback for the given Job name. The
	// callback is invoked when the Job is retrieved from the queue to be
	// performed. This would be used to initialize things such as database
	// connections.
	InitFunc(string, InitFunc)

	// Consume begins consuming jobs that have been submitted onto the queue.
	// This should only stop when the given Context is cancelled.
	Consume(context.Context) error

	// Produce places the given Job onto the end of the queue. This should
	// return the ID of the Job, if possible, this will vary depending on the
	// implementation being used.
	Produce(context.Context, Job) (string, error)
}

// Redis offers an implementation of the Queue interface using curlyq for
// producing/consuming from/to Redis.
type Redis struct {
	reg *InitRegistry
	log *log.Logger
	prd *curlyq.Producer
	con *curlyq.Consumer
}

// Memory offers an in-memory implementation of the Queue interface. This will
// queue up jobs in memory, and process them. This is ideal for jobs that aren't
// that consequential.
type Memory struct {
	n    int
	mu   *sync.Mutex
	wg   *sync.WaitGroup
	reg  *InitRegistry
	errh func(Job, error)
	jobs []Job
}

var (
	_ Queue = (*Redis)(nil)
	_ Queue = (*Memory)(nil)

	// ErrNilProducer should be returned when a Queue implementation is not
	// setup as a producer, and a call to Produce is made.
	ErrNilProducer = errors.New("nil queue producer")

	// ErrNilConsumer should be returned when a Queue implementation is not
	// setup as a consumer, and a call to Consume is made.
	ErrNilConsumer = errors.New("nil queue consumer")

	// ErrQueueNotExist is returned when a Job is dispatched to a non-existent
	// queue in a Set.
	ErrQueueNotExist = errors.New("queue does not exist")
)

func NewRedisConsumer(log *log.Logger, opts *curlyq.ConsumerOpts) *Redis {
	opts.Queue = queueName

	return &Redis{
		reg: NewInitRegistry(),
		log: log,
		con: curlyq.NewConsumer(opts),
	}
}

func NewRedisProducer(log *log.Logger, opts *curlyq.ProducerOpts) *Redis {
	opts.Queue = queueName

	return &Redis{
		reg: NewInitRegistry(),
		log: log,
		prd: curlyq.NewProducer(opts),
	}
}

// NewMemory returns a new in-memory Queue for Job processing with the given
// parallelism as defined by n, and the given error handler. The error handler
// is invoked whenever an underlying Job being processed on the queue returns
// an error from a Perform call.
func NewMemory(n int, errh func(Job, error)) *Memory {
	return &Memory{
		n:    n,
		mu:   &sync.Mutex{},
		wg:   &sync.WaitGroup{},
		reg:  NewInitRegistry(),
		errh: errh,
		jobs: make([]Job, 0),
	}
}

// NewInitRegistry returns a new registry for registered initialization
// functions against a given name.
func NewInitRegistry() *InitRegistry {
	return &InitRegistry{
		mu:  &sync.RWMutex{},
		fns: make(map[string]InitFunc),
	}
}

func NewSet() *Set {
	return &Set{
		mu:     &sync.RWMutex{},
		queues: make(map[string]Queue),
	}
}

// Register registers the given InitFunc against the given name. If the given
// name has already been registered, then this panics.
func (r *InitRegistry) Register(name string, fn InitFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.fns[name]; ok {
		panic("queue: init function already registered for job: " + name)
	}
	r.fns[name] = fn
}

// Get returns the InitFunc for the given name, along with whether a function
// has been registered against that name.
func (r *InitRegistry) Get(name string) (InitFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fn, ok := r.fns[name]
	return fn, ok
}

// Add adds the given Queue to the Set with the given name. This will panic if
// the given name has already been set.
func (s *Set) Add(name string, q Queue) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.queues[name]; ok {
		panic("queue: queue with name " + name + " already in set")
	}
	s.queues[name] = q
}

// InitFunc will register the given initialization function with the given name
// against all of the queues in the Set.
func (s *Set) InitFunc(name string, fn InitFunc) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, q := range s.queues {
		q.InitFunc(name, fn)
	}
}

// Produce will place the given Job onto the Queue with the given name in the
// Set. If the Queue cannot be found, then ErrQueueNotExist is returned as the
// error.
func (s *Set) Produce(ctx context.Context, name string, j Job) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q, ok := s.queues[name]

	if !ok {
		return "", ErrQueueNotExist
	}
	return q.Produce(ctx, j)
}

// InitFunc implementas the Queue interface.
func (m *Memory) InitFunc(name string, fn InitFunc) { m.reg.Register(name, fn) }

// Produce implements the Queue interface. This does not return anything for
// the ID of the Job.
func (m *Memory) Produce(ctx context.Context, j Job) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", errors.Err(err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.jobs = append(m.jobs, j)
	return "", nil
}

func (m *Memory) dequeue() (Job, bool) {
	if len(m.jobs) == 0 {
		return nil, false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	j := m.jobs[0]
	m.jobs = m.jobs[1:]

	return j, true
}

// Consume implements the Queue interface.
func (m *Memory) Consume(ctx context.Context) error {
	sem := make(chan struct{}, m.n)

	for i := 0; i < m.n; i++ {
		sem <- struct{}{}
	}

	for {
		select {
		case <-ctx.Done():
			m.wg.Wait()
			close(sem)
			return ctx.Err()
		case <-sem:
			j, ok := m.dequeue()

			if !ok {
				sem <- struct{}{}
				continue
			}

			m.wg.Add(1)

			go func() {
				defer func() {
					sem <- struct{}{}
					m.wg.Done()
				}()

				fn, ok := m.reg.Get(j.Name())

				if ok {
					fn(j)
				}

				if err := j.Perform(); err != nil {
					m.errh(j, err)
				}
			}()
		}
	}
}

// InitFunc implements the Queue interface.
func (c *Redis) InitFunc(name string, fn InitFunc) { c.reg.Register(name, fn) }

// Produce implements the Queue interface. The given Job is encoded into bytes
// using gob encoding. This returns the underlying Job ID from curlyq itself.
// If Redis has not been configured as a producer, then ErrNilProducer is
// returned.
func (r *Redis) Produce(ctx context.Context, j Job) (string, error) {
	var buf bytes.Buffer

	if r.prd == nil {
		return "", ErrNilProducer
	}

	if err := gob.NewEncoder(&buf).Encode(&j); err != nil {
		return "", errors.Err(err)
	}

	id, err := r.prd.PerformCtx(ctx, curlyq.Job{
		Data: buf.Bytes(),
	})
	return id, errors.Err(err)
}

func (r *Redis) handler(ctx context.Context, j0 curlyq.Job) error {
	defer func() {
		if v := recover(); v != nil {
			if err, ok := v.(error); ok {
				r.log.Error.Println(err.Error())
			}
			r.log.Error.Println(string(debug.Stack()))
		}
	}()

	if err := ctx.Err(); err != nil {
		return errors.Err(err)
	}

	var j Job

	if err := gob.NewDecoder(bytes.NewReader(j0.Data)).Decode(&j); err != nil {
		return errors.Err(err)
	}

	fn, ok := r.reg.Get(j.Name())

	if ok {
		fn(j)
	}
	return errors.Err(j.Perform())
}

// Consume implements the Queue interface. If Redis has not been configured as
// a consumer, then ErrNilConsumer is returned.
func (r *Redis) Consume(ctx context.Context) error {
	if r.con == nil {
		return ErrNilConsumer
	}
	return r.con.ConsumeCtx(ctx, r.handler)
}
