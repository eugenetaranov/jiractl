// Package tui holds jiractl's interactive components, built on Bubble Tea.
// Every component renders inline on stderr, so stdout only ever carries
// command results, and returns ErrCancelled on Esc or ErrInterrupted on
// Ctrl+C.
package tui

import (
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// ErrCancelled is returned when the user presses Esc.
var ErrCancelled = errors.New("cancelled")

// ErrInterrupted is returned when the user presses Ctrl+C. It wraps
// ErrCancelled, so errors.Is(err, ErrCancelled) covers both; use IsEsc to
// tell "go back / skip" from "stop everything".
var ErrInterrupted = fmt.Errorf("%w: interrupted", ErrCancelled)

// IsEsc reports whether err is a plain Esc (not Ctrl+C).
func IsEsc(err error) bool {
	return errors.Is(err, ErrCancelled) && !errors.Is(err, ErrInterrupted)
}

// IsTerminal reports whether interactive components can run.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stderr.Fd()))
}

// errNoTerminal is returned instead of starting a program without a TTY.
var errNoTerminal = errors.New("this needs an interactive terminal")

// run starts a program on stderr.
func run(m tea.Model) (tea.Model, error) {
	if !IsTerminal() {
		return nil, errNoTerminal
	}
	ensureWindowSize()
	final, err := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInput(os.Stdin)).Run()
	if errors.Is(err, tea.ErrInterrupted) {
		return final, ErrInterrupted
	}
	return final, err
}

// Styles use the terminal's 16 ANSI colors so they follow its theme;
// NO_COLOR is honored by Bubble Tea's color profile detection.
var (
	styleTitle    = lipgloss.NewStyle().Bold(true)
	styleCursor   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	styleSelected = lipgloss.NewStyle().Bold(true)
	styleDim      = lipgloss.NewStyle().Faint(true)
	styleError    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleDone     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

// doneLine is what a finished prompt leaves behind: "✓ Label: value".
func doneLine(label, value string) string {
	return styleDone.Render("✓") + " " + styleTitle.Render(label) + " " + value
}

// failLine is what a field that gave up leaves behind.
func failLine(label, problem string) string {
	return styleError.Render("✗") + " " + styleTitle.Render(label) + " " + styleError.Render(problem)
}

func errorsIsCancel(err error) bool {
	return errors.Is(err, ErrCancelled)
}

// keyIs reports whether msg is one of the given key presses.
func keyIs(msg tea.Msg, keys ...string) bool {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false
	}
	s := k.String()
	for _, want := range keys {
		if s == want {
			return true
		}
	}
	return false
}
