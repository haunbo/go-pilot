// Package xgo provides goroutine, panic recovery and parallel execution helpers.
package xgo

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/haunbo/go-pilot/pkg/xlog"
)

// Recover runs fn and converts a panic into an error. If onPanic is nil the
// panic is logged with the default logger.
func Recover(fn func(), onPanic func(any)) (err error) {
	if fn == nil {
		return errors.New("xgo: nil function")
	}
	defer func() {
		if rec := recover(); rec != nil {
			if recovered, ok := rec.(error); ok {
				err = recovered
			} else {
				err = fmt.Errorf("%v", rec)
			}
			if onPanic != nil {
				onPanic(rec)
				return
			}
			xlog.Error("goroutine panic", xlog.FieldErr(err), xlog.String("stack", string(debug.Stack())))
		}
	}()
	fn()
	return nil
}

// Go runs fn in a new goroutine and recovers panics.
func Go(fn func()) {
	go func() {
		_ = Recover(fn, nil)
	}()
}

// GoWithRecover runs fn in a new goroutine and calls onPanic on panic.
func GoWithRecover(fn func(), onPanic func(any)) {
	go func() {
		_ = Recover(fn, onPanic)
	}()
}

// DelayGo runs fn after delay in a new goroutine.
func DelayGo(delay time.Duration, fn func()) {
	go func() {
		if delay > 0 {
			time.Sleep(delay)
		}
		_ = Recover(fn, nil)
	}()
}

// Parallel runs all functions concurrently and waits for them to finish.
func Parallel(fns ...func()) {
	var wg sync.WaitGroup
	wg.Add(len(fns))
	for _, fn := range fns {
		fn := fn
		go func() {
			defer wg.Done()
			_ = Recover(fn, nil)
		}()
	}
	wg.Wait()
}

// RestrictParallel runs functions concurrently with at most limit in flight.
func RestrictParallel(limit int, fns ...func()) {
	if limit <= 0 || limit > len(fns) {
		limit = len(fns)
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	wg.Add(len(fns))
	for _, fn := range fns {
		fn := fn
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			_ = Recover(fn, nil)
		}()
	}
	wg.Wait()
}

// ParallelWithContext runs functions concurrently, skipping new functions once
// ctx is cancelled.
func ParallelWithContext(ctx context.Context, fns ...func()) {
	var wg sync.WaitGroup
	wg.Add(len(fns))
	for _, fn := range fns {
		fn := fn
		go func() {
			defer wg.Done()
			if ctx.Err() != nil {
				return
			}
			_ = Recover(fn, nil)
		}()
	}
	wg.Wait()
}

// ParallelWithError runs functions concurrently and joins all errors.
func ParallelWithError(fns ...func() error) error {
	errs := make([]error, len(fns))
	var wg sync.WaitGroup
	wg.Add(len(fns))
	for i, fn := range fns {
		i, fn := i, fn
		go func() {
			defer wg.Done()
			var err error
			if panicErr := Recover(func() { err = fn() }, nil); panicErr != nil {
				err = panicErr
			}
			errs[i] = err
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

// ParallelWithErrorContext stops launching work after ctx is cancelled and
// returns the first error or ctx.Err().
func ParallelWithErrorContext(ctx context.Context, fns ...func(context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errs := make([]error, len(fns))
	var wg sync.WaitGroup
	wg.Add(len(fns))
	for i, fn := range fns {
		i, fn := i, fn
		go func() {
			defer wg.Done()
			if err := ctx.Err(); err != nil {
				errs[i] = err
				return
			}
			var err error
			if panicErr := Recover(func() { err = fn(ctx) }, nil); panicErr != nil {
				err = panicErr
			}
			errs[i] = err
			if err != nil {
				cancel()
			}
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

// SerialWithError runs functions one by one and joins all errors.
func SerialWithError(fns ...func() error) error {
	errs := make([]error, 0, len(fns))
	for _, fn := range fns {
		if err := Recover(func() {
			errs = append(errs, fn())
		}, nil); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// SerialUntilError runs functions one by one and stops at the first error.
func SerialUntilError(fns ...func() error) error {
	for _, fn := range fns {
		var err error
		if panicErr := Recover(func() { err = fn() }, nil); panicErr != nil {
			return panicErr
		}
		if err != nil {
			return err
		}
	}
	return nil
}
