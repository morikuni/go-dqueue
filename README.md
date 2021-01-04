# go-dqueue

[![test](https://github.com/morikuni/go-dqueue/workflows/test/badge.svg?branch=main)](https://github.com/morikuni/go-dqueue/actions?query=branch%3Amain)
[![Go Reference](https://pkg.go.dev/badge/github.com/morikuni/go-dqueue.svg)](https://pkg.go.dev/github.com/morikuni/go-dqueue)
[![Go Report Card](https://goreportcard.com/badge/github.com/morikuni/go-dqueue)](https://goreportcard.com/report/github.com/morikuni/go-dqueue)
[![codecov](https://codecov.io/gh/morikuni/go-dqueue/branch/main/graph/badge.svg)](https://codecov.io/gh/morikuni/go-dqueue)

`dqueue` provides high performance delay queue with unlimited buffer in Go.

## Usage

Try it on [The Go Playground](https://play.golang.org/p/tf6TNCX8rRu).

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/morikuni/go-dqueue"
)

func Example() {
	queue := dqueue.New()
	queue.Push("hello", dqueue.Delay(time.Second))
	fmt.Println(time.Now())
	fmt.Println(queue.Pull(context.Background()))
	fmt.Println(time.Now())
	// Output:
	// 2009-11-10 23:00:00 +0000 UTC m=+0.000000001
	// hello <nil>
	// 2009-11-10 23:00:01 +0000 UTC m=+1.000000001
}
```

`queue.Pull` blocks until any item become available after the delay.

## Performance

```
pkg: github.com/morikuni/go-dqueue
BenchmarkQueue_Push          55974181      232 ns/op    125 B/op    2 allocs/op
BenchmarkQueue_Pull          21256920      700 ns/op      0 B/op    0 allocs/op
BenchmarkQueue_PushPull      27615370      436 ns/op     80 B/op    2 allocs/op
```

1,000,000+ ops/sec.