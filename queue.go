package queue

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

type Queue struct {
	mu    sync.Mutex
	queue *queue
	c     chan struct{}
}

func New() *Queue {
	return &Queue{
		queue: &queue{},
		c:     make(chan struct{}),
	}
}

func (q *Queue) Push(v interface{}, t time.Time) {
	q.mu.Lock()
	heap.Push(q.queue, &item{v, t})
	q.mu.Unlock()
	select {
	case q.c <- struct{}{}:
	default:
	}
}

func (q *Queue) Pull(ctx context.Context) (interface{}, error) {
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

		i := q.queue.peek().(*item)
		d := i.Timestamp.Sub(time.Now())
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

type queue []*item

var _ heap.Interface = (*queue)(nil)

func (q queue) Len() int {
	return len(q)
}

func (q queue) Less(i, j int) bool {
	return q[i].Timestamp.Before(q[j].Timestamp)
}

func (q queue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

func (q *queue) Push(x interface{}) {
	*q = append(*q, x.(*item))
}

func (q *queue) Pop() interface{} {
	n := len(*q)
	item := (*q)[n-1]
	(*q)[n-1] = nil // avoid memory leak
	*q = (*q)[:n-1]
	return item
}

func (q *queue) peek() interface{} {
	return (*q)[0]
}

type item struct {
	Value     interface{}
	Timestamp time.Time
}
