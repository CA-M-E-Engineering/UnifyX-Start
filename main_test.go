package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRestartDelay(t *testing.T) {
	delay := time.Duration(0)
	for _, seconds := range []int{3, 6, 12, 24, 30, 30} {
		delay = restartDelay(delay, time.Second)
		if delay != time.Duration(seconds)*time.Second {
			t.Fatalf("delay = %s, want %ds", delay, seconds)
		}
	}
	if got := restartDelay(delay, time.Minute); got != 3*time.Second {
		t.Fatalf("stable run should reset backoff: %s", got)
	}
}

func TestSuperviseReturnsLaunchFailureWithoutWatchdog(t *testing.T) {
	want := errors.New("browser failed")
	if got := supervise(context.Background(), func(context.Context) error { return want }, false); !errors.Is(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSuperviseStopsDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	launched := make(chan struct{})
	done := make(chan error, 1)
	calls := 0
	go func() {
		done <- supervise(ctx, func(context.Context) error {
			calls++
			close(launched)
			return errors.New("crash")
		}, true)
	}()
	<-launched
	select {
	case err := <-done:
		if err != nil || calls != 1 {
			t.Fatalf("err=%v, launches=%d", err, calls)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown waited for the restart timer")
	}
}

func TestSuperviseDoesNotLaunchWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := supervise(ctx, func(context.Context) error { t.Fatal("unexpected launch"); return nil }, true); err != nil {
		t.Fatal(err)
	}
}

func TestEnvBool(t *testing.T) {
	for _, value := range []string{"", "true", "false", "TRUE", "invalid"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("UNIFYX_TEST_BOOL", value)
			got, err := envBool("UNIFYX_TEST_BOOL")
			if (err != nil) != (value == "invalid") || got != (value == "true" || value == "TRUE") {
				t.Fatalf("value=%q, got=%v, error=%v", value, got, err)
			}
		})
	}
}
