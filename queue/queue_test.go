package queue

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func errh(t *testing.T) func(Job, error) {
	return func(j Job, err error) {
		t.Error(j.Name(), t)
	}
}

type Cancel struct {
	Cancel func()
}

type Sleep struct {
	D time.Duration
}

func (c Cancel) Name() string { return "cancel_job" }

func (c Cancel) Perform() error {
	c.Cancel()
	return nil
}

func (s Sleep) Name() string { return "sleep_job" }

func (s Sleep) Perform() error {
	fmt.Println("sleeping for", s.D)
	time.Sleep(s.D)
	return nil
}

func Test_Queue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mem := NewMemory(4, errh(t))
	mem.Produce(ctx, Sleep{time.Second*1})
	mem.Produce(ctx, Sleep{time.Second*2})
	mem.Produce(ctx, Sleep{time.Second*3})
	mem.Produce(ctx, Cancel{cancel})

	mem.Consume(ctx)
}
