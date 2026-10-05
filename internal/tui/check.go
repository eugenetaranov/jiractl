package tui

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// checkResultMsg carries the outcome of an async Check.
type checkResultMsg struct {
	seq int
	err error
}

// checker runs a field's Check off the UI goroutine, shows a spinner while
// it runs, and counts failures.
type checker struct {
	label    string // "Checking server"
	max      int    // give up after this many failures (0 = never)
	running  bool
	failures int
	seq      int
	spin     spinner.Model
}

func newChecker(label string, maxAttempts int) checker {
	if label == "" {
		label = "Checking"
	}
	return checker{
		label: label,
		max:   maxAttempts,
		spin:  spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(styleCursor)),
	}
}

// start runs check(…) in the background.
func (c *checker) start(check func() error) tea.Cmd {
	c.running = true
	c.seq++
	seq := c.seq
	return tea.Batch(c.spin.Tick, func() tea.Msg {
		return checkResultMsg{seq: seq, err: check()}
	})
}

// result records a finished check. It reports whether the result is current
// and whether the field should give up.
func (c *checker) result(msg checkResultMsg) (current, giveUp bool) {
	if msg.seq != c.seq {
		return false, false
	}
	c.running = false
	if msg.err != nil {
		c.failures++
		return true, c.max > 0 && c.failures >= c.max
	}
	return true, false
}

func (c *checker) update(msg tea.Msg) tea.Cmd {
	if !c.running {
		return nil
	}
	var cmd tea.Cmd
	c.spin, cmd = c.spin.Update(msg)
	return cmd
}

func (c *checker) view() string {
	return c.spin.View() + " " + styleDim.Render(c.label+"…")
}
