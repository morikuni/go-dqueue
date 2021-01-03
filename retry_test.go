package queue

import (
	"context"
	"testing"
)

func TestRetryQueue(t *testing.T) {
	q := NewRetryQueue()

	q.Push(1)

	ri, err := q.PullRetryItem(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	equal(t, ri.RetryCount(), 0)
	equal(t, ri.Value(), 1)

	ri.RetryAfter(0)

	ri, err = q.PullRetryItem(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	equal(t, ri.RetryCount(), 1)
	equal(t, ri.Value(), 1)
}
