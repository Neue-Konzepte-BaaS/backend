package services

import (
	"context"
	"sync"
)

// Dispatcher runs work off the request goroutine and lets shutdown wait for it
// to finish.
//
// Notification fan-out is deliberately not part of the request: net/smtp opens
// a fresh connection per message, so a broadcast to dozens of recipients would
// otherwise hold the HTTP response open for tens of seconds. The trade is that
// delivery becomes best-effort — nothing is retried, and failures are logged
// rather than reported to the caller.
type Dispatcher struct {
	wg  sync.WaitGroup
	sem chan struct{}
}

// NewDispatcher returns a Dispatcher running at most concurrency tasks at once.
// A concurrency below one is treated as one, so a misconfigured value degrades
// to serial delivery instead of deadlocking.
func NewDispatcher(concurrency int) *Dispatcher {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Dispatcher{sem: make(chan struct{}, concurrency)}
}

// Go runs fn in the background. It returns immediately, even when every slot is
// busy: the task blocks on the semaphore in its own goroutine rather than in
// the caller's.
func (d *Dispatcher) Go(fn func()) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()

		d.sem <- struct{}{}
		defer func() { <-d.sem }()

		fn()
	}()
}

// Wait blocks until every dispatched task has finished, or until ctx is done —
// whichever comes first, returning ctx.Err() in the latter case. The deadline
// matters because smtp.SendMail takes no context and has no timeout, so a hung
// relay would otherwise keep the process alive indefinitely.
func (d *Dispatcher) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
