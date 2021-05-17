// Package queue provides an in-memory implementation of a queue for the
// processing of jobs. Jobs placed on to this queue are simple functions.
package queue

import (
	"context"
	"sync"
)

type item struct {
	fn func() error
}

// Queue is an in-memory queue for background processing of lightweight jobs,
// such as sending of emails, and dispatching of webhooks.
type Queue struct {
	n      int
	acked  int
	failed int
	mu     *sync.Mutex
	wg     *sync.WaitGroup
	errh   func(error)
	items  []*item
}

// New creates a new in-memory Queue with the given parallelism, and error
// handler. The given error handler is called whenever a fatal error occurs
// during processing of jobs placed onto the queue.
func New(n int, errh func(error)) *Queue {
	return &Queue{
		n:     n,
		mu:    &sync.Mutex{},
		wg:    &sync.WaitGroup{},
		errh:  errh,
		items: make([]*item, 0),
	}
}

// Acked returns the number of jobs that have been retrieved from the queue for
// processing.
func (q *Queue) Acked() int { return q.acked }

// Failed returns the number of jobs that have failed execution.
func (q *Queue) Failed() int { return q.failed }

// Enqueue adds the given function on to the end of the queue.
func (q *Queue) Enqueue(fn func() error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = append(q.items, &item{
		fn: fn,
	})
}

// Dequeue returns the job from the front of the queue, and returns it. The bool
// that is returned denotes whether a job was retrieved from the queue.
func (q *Queue) Dequeue() (func() error, bool) {
	if len(q.items) == 0 {
		return nil, false
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	it := q.items[0]
	q.items = q.items[1:]

	return it.fn, true
}

// Run will begin processing jobs that are placed on to the queue. The given
// context will be used to cancel the processing of jobs.
func (q *Queue) Run(ctx context.Context) {
	sem := make(chan struct{}, q.n)

	for i := 0; i < q.n; i++ {
		sem <- struct{}{}
	}

	acked := make(chan struct{}, q.n)
	failed := make(chan struct{}, q.n)

	go func() {
		for {
			select {
			case _, ok := <-acked:
				if ok {
					q.acked++
				}
			case _, ok := <-failed:
				if ok {
					q.failed++
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			q.wg.Wait()
			close(acked)
			close(failed)
			close(sem)
			return
		case <-sem:
			fn, ok := q.Dequeue()

			if !ok {
				sem <- struct{}{}
				continue
			}

			q.wg.Add(1)

			go func() {
				defer func() {
					sem <- struct{}{}
					q.wg.Done()
				}()

				acked <- struct{}{}

				if err := fn(); err != nil {
					q.errh(err)
				}
			}()
		}
	}
}
