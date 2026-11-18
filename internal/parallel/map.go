// Package parallel provides a small bounded worker pool with deterministic
// result ordering.
package parallel

import (
	"context"
	"fmt"
	"runtime"
	"sync"
)

// Map runs fn over items with at most workers concurrent calls. Results are
// returned in input order. The first error cancels remaining work.
func Map[T any, R any](ctx context.Context, items []T, workers int, fn func(context.Context, int, T) (R, error)) ([]R, error) {
	out := make([]R, len(items))
	if len(items) == 0 {
		return out, nil
	}
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers > 16 {
			workers = 16
		}
	}
	if workers > 64 {
		workers = 64
	}
	if workers > len(items) {
		workers = len(items)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan int)
	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error
	fail := func(err error) {
		once.Do(func() {
			firstErr = err
			cancel()
		})
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				if ctx.Err() != nil {
					continue
				}
				func() {
					defer func() {
						if rec := recover(); rec != nil {
							fail(fmt.Errorf("parallel panic: %v", rec))
						}
					}()
					v, err := fn(ctx, idx, items[idx])
					if err != nil {
						fail(err)
						return
					}
					out[idx] = v
				}()
			}
		}()
	}
sendLoop:
	for i := range items {
		select {
		case jobs <- i:
		case <-ctx.Done():
			break sendLoop
		}
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return out, firstErr
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	return out, nil
}
