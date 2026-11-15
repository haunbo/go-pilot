package xgo

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRecover(t *testing.T) {
	if err := Recover(func() {}, nil); err != nil {
		t.Fatal(err)
	}
	if err := Recover(func() { panic("boom") }, func(any) {}); err == nil || err.Error() != "boom" {
		t.Fatalf("panic error = %v", err)
	}
}

func TestParallel(t *testing.T) {
	var count int64
	Parallel(
		func() { atomic.AddInt64(&count, 1) },
		func() { atomic.AddInt64(&count, 1) },
		func() { panic("recovered") },
	)
	if got := atomic.LoadInt64(&count); got != 2 {
		t.Fatalf("count = %d", got)
	}
}

func TestRestrictParallel(t *testing.T) {
	var current, peak int64
	RestrictParallel(2,
		func() { trackConcurrency(&current, &peak) },
		func() { trackConcurrency(&current, &peak) },
		func() { trackConcurrency(&current, &peak) },
		func() { trackConcurrency(&current, &peak) },
	)
	if peak > 2 {
		t.Fatalf("peak concurrency = %d", peak)
	}
}

func trackConcurrency(current, peak *int64) {
	n := atomic.AddInt64(current, 1)
	for {
		old := atomic.LoadInt64(peak)
		if n <= old || atomic.CompareAndSwapInt64(peak, old, n) {
			break
		}
	}
	time.Sleep(2 * time.Millisecond)
	atomic.AddInt64(current, -1)
}

func TestParallelWithError(t *testing.T) {
	err := ParallelWithError(
		func() error { return nil },
		func() error { return errors.New("first") },
		func() error { return errors.New("second") },
	)
	if err == nil || !errors.Is(err, err) {
		t.Fatalf("error = %v", err)
	}
}

func TestParallelWithErrorContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ParallelWithErrorContext(ctx, func(context.Context) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestSerial(t *testing.T) {
	var count int64
	err := SerialWithError(
		func() error { atomic.AddInt64(&count, 1); return nil },
		func() error { atomic.AddInt64(&count, 1); return errors.New("boom") },
		func() error { atomic.AddInt64(&count, 1); return nil },
	)
	if err == nil || count != 3 {
		t.Fatalf("err = %v, count = %d", err, count)
	}
	err = SerialUntilError(
		func() error { atomic.AddInt64(&count, 1); return nil },
		func() error { return errors.New("stop") },
		func() error { atomic.AddInt64(&count, 1); return nil },
	)
	if err == nil || count != 4 {
		t.Fatalf("err = %v, count = %d", err, count)
	}
}
