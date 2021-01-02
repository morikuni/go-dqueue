package queue

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

type Queue interface {
	Push(v interface{}, opts ...PushOption)
	Pull(ctx context.Context) (interface{}, error)
	Flush() []interface{}
}

type queue struct {
	mu    sync.Mutex
	queue *timeHeap
	c     chan struct{}
}

func New() Queue {
	return &queue{
		queue: &timeHeap{},
		c:     make(chan struct{}),
	}
}

func (q *queue) Push(v interface{}, opts ...PushOption) {
	conf := newPushConfig(opts)
	q.mu.Lock()
	heap.Push(q.queue, &item{v, conf.delayUntil})
	q.mu.Unlock()
	select {
	case q.c <- struct{}{}:
	default:
	}
}

func (q *queue) Pull(ctx context.Context) (interface{}, error) {
	for {
		q.mu.Lock()
		if q.queue.Len() == 0 {
			q.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-q.c:
				continue
			}
		}

		i := q.queue.peek()
		d := i.DelayUntil.Sub(time.Now())
		if d > 0 {
			q.mu.Unlock()
			t := time.NewTimer(d)
			select {
			case <-ctx.Done():
				t.Stop()
				return nil, ctx.Err()
			case <-q.c:
				t.Stop()
				continue
			case <-t.C:
				continue
			}
		}

		item := heap.Pop(q.queue).(*item)
		q.mu.Unlock()
		return item.Value, nil
	}
}

func (q *queue) Flush() []interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	is := make([]interface{}, 0, q.queue.Len())
	for q.queue.Len() > 0 {
		i := heap.Pop(q.queue).(*item)
		is = append(is, i.Value)
	}

	return is
}

type timeHeap []*item

var _ heap.Interface = (*timeHeap)(nil)

func (q timeHeap) Len() int {
	return len(q)
}

func (q timeHeap) Less(i, j int) bool {
	return q[i].DelayUntil.Before(q[j].DelayUntil)
}

func (q timeHeap) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

func (q *timeHeap) Push(x interface{}) {
	*q = append(*q, x.(*item))
}

func (q *timeHeap) Pop() interface{} {
	n := len(*q)
	item := (*q)[n-1]
	(*q)[n-1] = nil // avoid memory leak
	*q = (*q)[:n-1]
	return item
}

func (q *timeHeap) peek() *item {
	return (*q)[0]
}

type item struct {
	Value      interface{}
	DelayUntil time.Time
}

type PushOption interface {
	apply(c *pushConfig)
}

type pushOptionFunc func(c *pushConfig)

func (f pushOptionFunc) apply(c *pushConfig) {
	f(c)
}

func Delay(d time.Duration) PushOption {
	return pushOptionFunc(func(c *pushConfig) {
		c.delayUntil = time.Now().Add(d)
	})
}

func DelayUntil(t time.Time) PushOption {
	return pushOptionFunc(func(c *pushConfig) {
		c.delayUntil = t
	})
}

type pushConfig struct {
	delayUntil time.Time
}

func newPushConfig(opts []PushOption) *pushConfig {
	c := new(pushConfig)
	for _, o := range opts {
		o.apply(c)
	}
	if c.delayUntil.IsZero() {
		c.delayUntil = time.Now()
	}
	return c
}
