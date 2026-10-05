package tui

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ReviewRow is one label/value line of a review table.
type ReviewRow struct {
	Label, Value string
}

// Review shows a table of rows under title and asks prompt with the given
// single-key choices (e.g. "yedn"; Enter picks the first). After y or n the
// table stays on screen; after other keys it is cleared, since the caller
// will show an updated review.
func Review(title string, rows []ReviewRow, prompt, choices string) (byte, error) {
	final, err := run(&reviewModel{title: title, rows: rows, choice: choiceModel{label: prompt, valid: strings.ToLower(choices)}})
	if err != nil {
		return 0, err
	}
	m := final.(*reviewModel)
	return m.choice.chosen, m.choice.err
}

type reviewModel struct {
	title  string
	rows   []ReviewRow
	choice choiceModel
}

func (m *reviewModel) Init() tea.Cmd { return nil }

func (m *reviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cm, cmd := m.choice.Update(msg)
	m.choice = *cm.(*choiceModel)
	return m, cmd
}

const reviewMaxLines = 6

func (m *reviewModel) table() string {
	width := 0
	for _, r := range m.rows {
		width = max(width, lipgloss.Width(r.Label))
	}
	labelStyle := lipgloss.NewStyle().Width(width + 2).Foreground(lipgloss.Color("6"))
	var b strings.Builder
	b.WriteString(styleTitle.Render(m.title) + "\n")
	for _, r := range m.rows {
		lines := strings.Split(r.Value, "\n")
		shown := lines
		if len(lines) > reviewMaxLines {
			shown = lines[:reviewMaxLines]
		}
		for i, line := range shown {
			label := ""
			if i == 0 {
				label = r.Label + ":"
			}
			b.WriteString("  " + labelStyle.Render(label) + line + "\n")
		}
		if len(lines) > reviewMaxLines {
			b.WriteString("  " + labelStyle.Render("") + styleDim.Render("… "+strconv.Itoa(len(lines)-reviewMaxLines)+" more lines") + "\n")
		}
	}
	return b.String()
}

func (m *reviewModel) View() tea.View {
	if m.choice.finished {
		if m.choice.err == nil && (m.choice.chosen == 'y' || m.choice.chosen == 'n') {
			return tea.NewView(m.table() + m.choice.View().Content)
		}
		return tea.NewView("")
	}
	return tea.NewView(m.table() + "\n" + m.choice.View().Content)
}
