package xretry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo(t *testing.T) {
	count := 0
	err := Do(context.Background(), 3, time.Millisecond, func(context.Context) error {
		count++
		if count < 3 {
			return errors.New("retry")
		}
		return nil
	})
	if err != nil || count != 3 {
		t.Fatalf("err = %v, count = %d", err, count)
	}
}

func TestDoExhausted(t *testing.T) {
	count := 0
	want := errors.New("always")
	err := Do(context.Background(), 2, time.Millisecond, func(context.Context) error {
		count++
		return want
	})
	if !errors.Is(err, want) || count != 2 {
		t.Fatalf("err = %v, count = %d", err, count)
	}
}

func TestDoContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Do(ctx, 3, time.Second, func(context.Context) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestDoValue(t *testing.T) {
	value, err := DoValue(context.Background(), 2, 0, func(context.Context) (int, error) {
		return 42, nil
	})
	if err != nil || value != 42 {
		t.Fatalf("value = %d, err = %v", value, err)
	}
}
