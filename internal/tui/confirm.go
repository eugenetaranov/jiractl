package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Confirm asks a yes/no question answered with a single key; Enter picks
// the default.
func Confirm(label string, defaultYes bool) (bool, error) {
	valid := "yn"
	if !defaultYes {
		valid = "ny"
	}
	choice, err := Choice(label, valid)
	return choice == 'y', err
}

// Choice asks for one of the letters in valid (e.g. "yedn"), answered with a
// single key. Enter picks the first letter, which is shown upper-case.
func Choice(label, valid string) (byte, error) {
	final, err := run(&choiceModel{label: label, valid: strings.ToLower(valid)})
	if err != nil {
		return 0, err
	}
	m := final.(*choiceModel)
	return m.chosen, m.err
}

type choiceModel struct {
	label    string
	valid    string
	chosen   byte
	err      error
	finished bool
}

func (m *choiceModel) Init() tea.Cmd { return nil }

func (m *choiceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch s := k.String(); {
	case s == "ctrl+c":
		m.err, m.finished = ErrInterrupted, true
		return m, tea.Quit
	case s == "esc":
		m.err, m.finished = ErrCancelled, true
		return m, tea.Quit
	case s == "enter":
		m.chosen, m.finished = m.valid[0], true
		return m, tea.Quit
	case len(s) == 1 && strings.Contains(m.valid, strings.ToLower(s)):
		m.chosen, m.finished = strings.ToLower(s)[0], true
		return m, tea.Quit
	}
	return m, nil
}

func (m *choiceModel) hint() string {
	parts := make([]string, len(m.valid))
	for i := range m.valid {
		parts[i] = string(m.valid[i])
	}
	parts[0] = strings.ToUpper(parts[0])
	return "[" + strings.Join(parts, "/") + "]"
}

func (m *choiceModel) View() tea.View {
	if m.finished {
		if m.err != nil {
			return tea.NewView("")
		}
		answer := string(m.chosen)
		switch m.chosen {
		case 'y':
			answer = "yes"
		case 'n':
			answer = "no"
		}
		return tea.NewView(doneLine(m.label, answer) + "\n")
	}
	return tea.NewView(styleTitle.Render(m.label) + " " + styleDim.Render(m.hint()) + " ")
}
