package tui

import (
	"fmt"
	"os"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// Wait shows a spinner labelled "Loading <label>..." until done is closed.
// Without a terminal it prints one line instead.
func Wait(label string, done <-chan struct{}) {
	if !IsTerminal() {
		fmt.Fprintf(os.Stderr, "Loading %s...\n", label)
		<-done
		return
	}
	s := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(styleCursor))
	final, err := run(&waitModel{label: label, done: done, spin: s})
	if m, ok := final.(*waitModel); (ok && m.interrupted) || err == ErrInterrupted {
		// Waiting can't be cancelled midway, so Ctrl+C stops jiractl; the
		// program has already restored the terminal at this point.
		fmt.Fprintln(os.Stderr, "Cancelled.")
		os.Exit(130)
	}
}

type doneMsg struct{}

type waitModel struct {
	label       string
	interrupted bool
	done        <-chan struct{}
	spin        spinner.Model
	finished    bool
}

func (m *waitModel) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, func() tea.Msg {
		<-m.done
		return doneMsg{}
	})
}

func (m *waitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case doneMsg:
		m.finished = true
		return m, tea.Quit
	case tea.KeyPressMsg:
		if keyIs(msg, "ctrl+c") {
			m.interrupted, m.finished = true, true
			return m, tea.Quit
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	return m, cmd
}

func (m *waitModel) View() tea.View {
	if m.finished {
		return tea.NewView("")
	}
	return tea.NewView(m.spin.View() + " Loading " + m.label + "...")
}
