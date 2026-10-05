package cmd

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
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

	tty := term.IsTerminal(int(os.Stderr.Fd()))
	frames := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	if !tty {
		fmt.Fprintf(os.Stderr, "Loading %s...\n", label)
	}
	for i := 0; ; i++ {
		select {
		case <-p.done:
			if tty {
				fmt.Fprint(os.Stderr, "\r\033[K")
			}
			return p.val, p.err
		case <-tick.C:
			if tty {
				fmt.Fprintf(os.Stderr, "\r%c Loading %s...", frames[i%len(frames)], label)
			}
		}
	}
}
