package services

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatcher_WaitBlocksUntilTasksFinish(t *testing.T) {
	// This is the guarantee shutdown relies on: a notification that was queued
	// before SIGTERM must still be delivered afterwards.
	d := NewDispatcher(2)

	var done atomic.Int32
	release := make(chan struct{})
	for range 3 {
		d.Go(func() {
			<-release
			done.Add(1)
		})
	}

	if got := done.Load(); got != 0 {
		t.Fatalf("completed %d tasks before releasing them, want 0", got)
	}

	close(release)
	if err := d.Wait(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := done.Load(); got != 3 {
		t.Errorf("completed = %d, want 3", got)
	}
}

func TestDispatcher_WaitGivesUpWhenTheContextIsDone(t *testing.T) {
	// smtp.SendMail has no timeout, so Wait must be able to abandon a task that
	// never returns rather than keep the process alive forever.
	d := NewDispatcher(1)

	stuck := make(chan struct{})
	t.Cleanup(func() { close(stuck) })
	d.Go(func() { <-stuck })

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := d.Wait(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want context.DeadlineExceeded", err)
	}
}

func TestDispatcher_RespectsConcurrencyLimit(t *testing.T) {
	const limit = 2
	d := NewDispatcher(limit)

	var (
		mu      sync.Mutex
		running int
		peak    int
	)
	for range 8 {
		d.Go(func() {
			mu.Lock()
			running++
			peak = max(peak, running)
			mu.Unlock()

			time.Sleep(5 * time.Millisecond)

			mu.Lock()
			running--
			mu.Unlock()
		})
	}

	if err := d.Wait(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if peak > limit {
		t.Errorf("peak concurrency = %d, want at most %d", peak, limit)
	}
}

func TestDispatcher_GoDoesNotBlockTheCaller(t *testing.T) {
	// Handlers call Go on the request goroutine, so a saturated dispatcher must
	// not stall the HTTP response.
	d := NewDispatcher(1)

	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	d.Go(func() { <-block })

	returned := make(chan struct{})
	go func() {
		d.Go(func() {})
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Error("Go blocked while every slot was busy")
	}
}

func TestNewDispatcher_ZeroConcurrencyStillRuns(t *testing.T) {
	// A misconfigured value must degrade to serial delivery, not deadlock.
	d := NewDispatcher(0)

	var ran atomic.Bool
	d.Go(func() { ran.Store(true) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := d.Wait(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ran.Load() {
		t.Error("task did not run")
	}
}
