// Package xretry provides context-aware retry helpers with exponential backoff
// and jitter.
package xretry

import (
	"context"
	"math/rand/v2"
	"time"
)

const maxDelay = 30 * time.Second

// Do calls fn until it succeeds, attempts is exhausted or ctx is cancelled.
// attempts is the total number of calls (minimum 1). initialDelay is the first
// backoff duration; it doubles after each failure up to 30s.
func Do(ctx context.Context, attempts int, initialDelay time.Duration, fn func(context.Context) error) error {
	if fn == nil {
		return nil
	}
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	delay := initialDelay
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := fn(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt == attempts {
			break
		}
		if err := sleep(ctx, backoff(delay)); err != nil {
			return err
		}
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
	return lastErr
}

// DoValue is the generic variant of Do.
func DoValue[T any](ctx context.Context, attempts int, initialDelay time.Duration, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	var value T
	err := Do(ctx, attempts, initialDelay, func(ctx context.Context) error {
		v, err := fn(ctx)
		if err != nil {
			return err
		}
		value = v
		return nil
	})
	if err != nil {
		return zero, err
	}
	return value, nil
}

func backoff(delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}
	jitterMax := int64(delay / 2)
	if jitterMax <= 0 {
		return delay
	}
	return delay/2 + time.Duration(rand.Int64N(jitterMax+1))
}

func sleep(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
