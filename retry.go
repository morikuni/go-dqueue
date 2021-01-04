package queue

import (
	"context"
	"math/rand"
	"sync/atomic"
	"time"
)

type RetryQueue struct {
	queue *Queue
}

func NewRetryQueue() *RetryQueue {
	return &RetryQueue{New()}
}

func (q *RetryQueue) Push(v interface{}, opts ...PushOption) {
	q.queue.Push(&RetryItem{0, v, q.queue, 0}, opts...)
}

// Pull always returns *RetryItem.
func (q *RetryQueue) Pull(ctx context.Context) (interface{}, error) {
	return q.queue.Pull(ctx)
}

func (q *RetryQueue) PullRetryItem(ctx context.Context) (*RetryItem, error) {
	v, err := q.queue.Pull(ctx)
	if err != nil {
		return nil, err
	}

	return v.(*RetryItem), nil
}

func (q *RetryQueue) Flush() []interface{} {
	ris := q.queue.Flush()
	vs := make([]interface{}, len(ris))
	for i, ri := range ris {
		vs[i] = ri.(*RetryItem).value
	}
	return vs
}

type RetryItem struct {
	retryCount int
	value      interface{}
	queue      *Queue
	done       uint32
}

func (ri *RetryItem) Value() interface{} {
	return ri.value
}

func (ri *RetryItem) RetryCount() int {
	return ri.retryCount
}

func (ri *RetryItem) Retry() {
	const (
		maxBackoff  = 10 * time.Minute
		baseBackoff = 50 * time.Millisecond
	)

	backoff := baseBackoff
	for i := 0; i < ri.retryCount; i++ {
		backoff *= 4
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	ri.RetryAfter(time.Duration(float64(backoff) * rand.Float64()))
}

func (ri *RetryItem) RetryAfter(d time.Duration) {
	if atomic.CompareAndSwapUint32(&ri.done, 0, 1) {
		ri.retryCount++
		ri.queue.Push(ri, Delay(d))
	}
}
