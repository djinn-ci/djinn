package queue

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func errh(t *testing.T) func(error) {
	return func(err error) {
		t.Error(t)
	}
}

func sleep(d time.Duration) func() error {
	return func() error {
		fmt.Println("sleeping for", d)
		time.Sleep(d)
		return nil
	}
}

func doCancel(fn func()) func() error {
	return func() error {
		fn()
		return nil
	}
}

func Test_Queue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := New(4, errh(t))
	q.Enqueue(sleep(time.Second*1))
	q.Enqueue(sleep(time.Second*2))
	q.Enqueue(sleep(time.Second*3))
	q.Enqueue(doCancel(cancel))

	q.Run(ctx)

	if acked := q.Acked(); acked != 4 {
		t.Errorf("unexpected acked, expected=%d, got=%d\n", 4, acked)
	}

	if failed := q.Failed(); failed != 0 {
		t.Errorf("unexpected failed, expected=%d, got=%d\n", 0, failed)
	}
}
