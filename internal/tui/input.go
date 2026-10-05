package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// InputOptions configures Input and Secret.
type InputOptions struct {
	// Default is returned when the user presses Enter on an empty field.
	Default string
	// Required rejects an empty answer (when there is no Default).
	Required bool
	// Hint is shown dimmed after the prompt, e.g. "(leave empty to keep)".
	Hint string
}

// Input asks for one line of text.
func Input(label string, opts InputOptions) (string, error) {
	return runInput(label, opts, false)
}

// Secret asks for one line without echoing it.
func Secret(label string, opts InputOptions) (string, error) {
	return runInput(label, opts, true)
}

func runInput(label string, opts InputOptions, secret bool) (string, error) {
	final, err := run(newInputModel(label, opts, secret))
	if err != nil {
		return "", err
	}
	m := final.(*inputModel)
	return m.value, m.err
}

type inputModel struct {
	label    string
	opts     InputOptions
	secret   bool
	input    textinput.Model
	value    string
	problem  string
	err      error
	finished bool
}

func newInputModel(label string, opts InputOptions, secret bool) *inputModel {
	ti := textinput.New()
	ti.Prompt = ""
	ti.SetWidth(60)
	if secret {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'
	}
	ti.Focus()
	return &inputModel{label: label, opts: opts, secret: secret, input: ti}
}

func (m *inputModel) Init() tea.Cmd { return textinput.Blink }

func (m *inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch {
	case keyIs(msg, "ctrl+c"):
		m.err, m.finished = ErrInterrupted, true
		return m, tea.Quit
	case keyIs(msg, "esc"):
		m.err, m.finished = ErrCancelled, true
		return m, tea.Quit
	case keyIs(msg, "enter"):
		v := strings.TrimSpace(m.input.Value())
		if v == "" {
			v = m.opts.Default
		}
		if v == "" && m.opts.Required {
			m.problem = "This field is required"
			return m, nil
		}
		m.value, m.finished = v, true
		return m, tea.Quit
	}
	if _, ok := msg.(tea.KeyPressMsg); ok {
		m.problem = ""
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *inputModel) View() tea.View {
	if m.finished {
		if m.err != nil {
			return tea.NewView("")
		}
		shown := m.value
		if m.secret {
			shown = strings.Repeat("•", min(len(m.value), 12))
			if m.value == "" {
				shown = styleDim.Render("(unchanged)")
			}
		}
		return tea.NewView(doneLine(m.label+":", shown) + "\n")
	}
	head := styleTitle.Render(m.label + ":")
	if m.opts.Default != "" && !m.secret {
		head += styleDim.Render(" [" + m.opts.Default + "]")
	}
	if m.opts.Hint != "" {
		head += styleDim.Render(" " + m.opts.Hint)
	}
	s := head + " " + m.input.View()
	if m.problem != "" {
		s += "\n" + styleError.Render("  "+m.problem)
	}
	return tea.NewView(s)
}
