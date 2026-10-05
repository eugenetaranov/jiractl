package tui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
)

// Textarea asks for multi-line text: Enter adds a line, Ctrl+D finishes,
// Ctrl+E continues in $EDITOR, Esc cancels.
func Textarea(label, initial string) (string, error) {
	final, err := run(newTextareaModel(label, initial))
	if err != nil {
		return "", err
	}
	m := final.(*textareaModel)
	return m.value, m.err
}

type editorDoneMsg struct {
	text string
	err  error
}

type textareaModel struct {
	label    string
	area     textarea.Model
	value    string
	problem  string
	err      error
	finished bool
}

func newTextareaModel(label, initial string) *textareaModel {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.Prompt = "┃ "
	ta.SetHeight(6)
	ta.SetWidth(76)
	// Ctrl+D and Ctrl+E are ours (finish, editor); keep their textarea
	// meanings on Delete and End.
	ta.KeyMap.DeleteCharacterForward = key.NewBinding(key.WithKeys("delete"))
	ta.KeyMap.LineEnd = key.NewBinding(key.WithKeys("end"))
	ta.SetValue(initial)
	ta.Focus()
	return &textareaModel{label: label, area: ta}
}

func (m *textareaModel) Init() tea.Cmd { return textarea.Blink }

func (m *textareaModel) finish() (tea.Model, tea.Cmd) {
	m.value = strings.TrimRight(m.area.Value(), "\n ")
	m.finished = true
	return m, tea.Quit
}

func (m *textareaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.area.SetWidth(max(msg.Width-2, 20))
		return m, nil
	case editorDoneMsg:
		if msg.err != nil {
			m.problem = msg.err.Error()
			return m, nil
		}
		m.area.SetValue(msg.text)
		return m.finish()
	}
	switch {
	case keyIs(msg, "ctrl+c"):
		m.err, m.finished = ErrInterrupted, true
		return m, tea.Quit
	case keyIs(msg, "esc"):
		m.err, m.finished = ErrCancelled, true
		return m, tea.Quit
	case keyIs(msg, "ctrl+d"):
		return m.finish()
	case keyIs(msg, "ctrl+e"):
		session, err := newEditorSession(m.area.Value())
		if err != nil {
			m.problem = err.Error()
			return m, nil
		}
		return m, tea.ExecProcess(session.cmd, func(runErr error) tea.Msg {
			text, err := session.result(runErr)
			return editorDoneMsg{text, err}
		})
	}
	var cmd tea.Cmd
	m.area, cmd = m.area.Update(msg)
	return m, cmd
}

func (m *textareaModel) View() tea.View {
	if m.finished {
		if m.err != nil {
			return tea.NewView("")
		}
		lines := strings.Count(m.value, "\n") + 1
		summary := firstLine(m.value)
		switch {
		case m.value == "":
			summary = styleDim.Render("(empty)")
		case lines > 1:
			summary += styleDim.Render(" … (" + strconv.Itoa(lines) + " lines)")
		}
		return tea.NewView(doneLine(m.label+":", summary) + "\n")
	}
	s := styleTitle.Render(m.label) + " " + styleDim.Render("Ctrl+D to finish · Ctrl+E for $EDITOR · Esc to cancel") + "\n" + m.area.View()
	if m.problem != "" {
		s += "\n" + styleError.Render("  "+m.problem)
	}
	return tea.NewView(s)
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return truncate(line, 60)
}
