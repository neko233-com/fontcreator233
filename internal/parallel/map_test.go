package parallel

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestMapPreservesOrder(t *testing.T) {
	items := []int{0, 1, 2, 3, 4, 5, 6, 7}
	out, err := Map(context.Background(), items, 4, func(_ context.Context, i, v int) (string, error) {
		if v%2 == 0 {
			time.Sleep(5 * time.Millisecond)
		}
		return string(rune('a' + v)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range out {
		if v != string(rune('a'+i)) {
			t.Fatalf("out[%d] = %q", i, v)
		}
	}
}

func TestMapCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var started atomic.Int32
	_, err := Map(ctx, []int{1, 2, 3, 4, 5}, 2, func(ctx context.Context, i, v int) (int, error) {
		started.Add(1)
		cancel()
		<-ctx.Done()
		return 0, ctx.Err()
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestMapPanic(t *testing.T) {
	_, err := Map(context.Background(), []int{1, 2, 3}, 2, func(_ context.Context, i, v int) (int, error) {
		if v == 2 {
			panic("boom")
		}
		return v, nil
	})
	if err == nil || !stringsContains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestMapWorkersClamped(t *testing.T) {
	out, err := Map(context.Background(), []int{1}, 0, func(_ context.Context, i, v int) (int, error) { return v * 2, nil })
	if err != nil || len(out) != 1 || out[0] != 2 {
		t.Fatalf("out=%v err=%v", out, err)
	}
}

func stringsContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
