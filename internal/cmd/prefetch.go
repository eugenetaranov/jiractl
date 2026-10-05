package cmd

import (
	"time"

	"github.com/eugenetaranov/jiractl/internal/tui"
)

// pending is a value being fetched in the background. Prompts that need it
// call wait; prompts that don't never block on the network.
type pending[T any] struct {
	done chan struct{}
	val  T
	err  error
}

func fetch[T any](f func() (T, error)) *pending[T] {
	p := &pending[T]{done: make(chan struct{})}
	go func() {
		defer close(p.done)
		p.val, p.err = f()
	}()
	return p
}

// ready wraps a value that is already known.
func ready[T any](v T) *pending[T] {
	p := &pending[T]{done: make(chan struct{}), val: v}
	close(p.done)
	return p
}

// spinnerDelay is how long a wait stays silent before a spinner appears.
const spinnerDelay = 300 * time.Millisecond

// wait blocks until the value arrives. After spinnerDelay it shows a spinner
// with label on stderr.
func (p *pending[T]) wait(label string) (T, error) {
	select {
	case <-p.done:
		return p.val, p.err
	case <-time.After(spinnerDelay):
	}
	tui.Wait(label, p.done)
	return p.val, p.err
}

// waitFor returns the value if it arrives within d; ok is false otherwise.
func (p *pending[T]) waitFor(d time.Duration) (val T, err error, ok bool) {
	select {
	case <-p.done:
		return p.val, p.err, true
	case <-time.After(d):
		return val, nil, false
	}
}
