package queue

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestQueue(t *testing.T) {
	q := New()

	now := time.Now()
	result := make(chan int, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	pull := func(q *Queue) {
		v, err := q.Pull(ctx)
		if err != nil {
			return
		}
		result <- v.(int)
	}
	push := func(q *Queue, v int, ts time.Time) {
		q.Push(v, DelayUntil(ts))
	}

	push(q, 5, now.Add(3*time.Second))

	c1 := make(chan struct{})
	c2 := make(chan struct{})
	go func() {
		pull(q)
		pull(q)
		close(c1)
	}()
	go func() {
		pull(q)
		pull(q)
		pull(q)
		close(c2)
	}()

	time.Sleep(100 * time.Millisecond)
	push(q, 3, now.Add(time.Second))
	time.Sleep(100 * time.Millisecond)
	push(q, 4, now.Add(2*time.Second))
	push(q, 1, now.Add(-time.Second))
	push(q, 2, now)

	select {
	case <-ctx.Done():
	case <-c1:
	}
	select {
	case <-ctx.Done():
	case <-c2:
	}
	close(result)

	for i := 0; i < 5; i++ {
		var v int
		select {
		case <-ctx.Done():
		case v = <-result:
		}
		if v != i+1 {
			t.Errorf("want %d got %d", i+1, v)
		}
	}
}

func BenchmarkQueue_Push(b *testing.B) {
	q := New()

	for i := 0; i < b.N; i++ {
		q.Push(1)
	}
}

func BenchmarkQueue_Pull(b *testing.B) {
	q := New()

	for i := 0; i < b.N; i++ {
		q.Push(1)
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Pull(ctx)
	}
}

func BenchmarkQueue_PushPull(b *testing.B) {
	q := New()

	c := make(chan struct{})
	go func() {
		for i := 0; i < b.N; i++ {
			q.Push(1)
		}
		close(c)
	}()

	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		q.Pull(ctx)
	}

	<-c
}

func TestQueuePBT(t *testing.T) {
	const (
		maxDelay  = 100
		allowDiff = 1
	)
	parallelism := gen.IntRange(1, 10)
	delays := gen.SliceOf(gen.UIntRange(0, maxDelay))

	p := gopter.DefaultTestParameters()
	p.MinSuccessfulTests = 100
	p.MaxShrinkCount = 100
	properties := gopter.NewProperties(nil)
	properties.Property("query parameter", prop.ForAll(func(pPush, pPull int, delays []uint) bool {
		delayChan := make(chan uint, len(delays))
		for _, d := range delays {
			delayChan <- d
		}

		start := make(chan struct{})
		stopPush := make(chan struct{})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := time.Now()

		q := New()
		for i := 0; i < pPush; i++ {
			go func() {
				<-start
				for {
					select {
					case d := <-delayChan:
						q.Push(d, Delay((50+time.Duration(d))*time.Millisecond))
					case <-stopPush:
						return
					}
				}
			}()
		}
		go func() {
			for {
				time.Sleep(10 * time.Millisecond)
				if len(delayChan) == 0 {
					close(stopPush)
					return
				}
			}
		}()

		resultChan := make(chan uint, len(delays))
		for i := 0; i < pPull; i++ {
			go func() {
				<-start
				for {
					d, err := q.Pull(context.Background())
					if err != nil {
						return
					}
					resultChan <- d.(uint)
				}
			}()
		}
		go func() {
			for {
				time.Sleep(10 * time.Millisecond)
				if len(resultChan) == len(delays) {
					cancel()
					return
				}
			}
		}()

		close(start)

		<-stopPush
		<-ctx.Done()

		sort.Slice(delays, func(i, j int) bool {
			return delays[i] < delays[j]
		})

		d := time.Now().Sub(now)
		if d > 2*maxDelay*time.Millisecond {
			t.Log("over max delay")
			return false
		}

		var result []uint
		for i := 0; i < len(delays); i++ {
			result = append(result, <-resultChan)
		}

		if len(result) != len(delays) {
			t.Logf("diff:\n%v\n%v", result, delays)
			return false
		}

		for i := range result {
			d := int(result[i]) - int(delays[i])
			if d < 0 {
				d = -d
			}
			if d > allowDiff { // sometimes
				t.Logf("diff:\n%v\n%v", result, delays)
				return false
			}
		}

		return true
	},
		parallelism,
		parallelism,
		delays,
	))

	properties.TestingRun(t)
}
