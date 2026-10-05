package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// SearchOptions configures SearchSelect.
type SearchOptions struct {
	Header string
	// Fixed rows are always listed first (e.g. "Skip: create without epic").
	Fixed []string
	// Search returns the rows matching query. It runs in the background,
	// Debounce after the user stops typing; results for an older query are
	// dropped.
	Search   func(query string) ([]string, error)
	Debounce time.Duration
	MaxRows  int
}

// SearchResult is what the user chose: a fixed row (Fixed >= 0) or a search
// result (Index >= 0) from the results for Query.
type SearchResult struct {
	Fixed int
	Index int
	Query string
}

// SearchSelect shows a list that is searched live as the user types.
func SearchSelect(opts SearchOptions) (SearchResult, error) {
	final, err := run(newSearchModel(opts))
	if err != nil {
		return SearchResult{Fixed: -1, Index: -1}, err
	}
	m := final.(*searchModel)
	return m.result, m.err
}

type (
	debounceMsg     struct{ seq int }
	searchResultMsg struct {
		seq   int
		query string
		items []string
		err   error
	}
)

type searchModel struct {
	opts     SearchOptions
	input    textinput.Model
	spin     spinner.Model
	seq      int
	loading  bool
	query    string // query the current results belong to
	results  []string
	problem  string
	cursor   int
	offset   int
	result   SearchResult
	err      error
	finished bool
}

func newSearchModel(opts SearchOptions) *searchModel {
	if opts.Debounce <= 0 {
		opts.Debounce = 250 * time.Millisecond
	}
	if opts.MaxRows <= 0 {
		opts.MaxRows = 12
	}
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "type to search Jira"
	ti.SetWidth(40)
	ti.Focus()
	return &searchModel{
		opts:   opts,
		input:  ti,
		spin:   spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(styleCursor)),
		result: SearchResult{Fixed: -1, Index: -1},
	}
}

func (m *searchModel) search(seq int, query string) tea.Cmd {
	m.loading = true
	return tea.Batch(m.spin.Tick, func() tea.Msg {
		items, err := m.opts.Search(query)
		return searchResultMsg{seq: seq, query: query, items: items, err: err}
	})
}

func (m *searchModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.search(m.seq, ""))
}

func (m *searchModel) total() int { return len(m.opts.Fixed) + len(m.results) }

func (m *searchModel) row(i int) string {
	if i < len(m.opts.Fixed) {
		return m.opts.Fixed[i]
	}
	return m.results[i-len(m.opts.Fixed)]
}

func (m *searchModel) finish(err error) (tea.Model, tea.Cmd) {
	m.err, m.finished = err, true
	return m, tea.Quit
}

func (m *searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.input.SetWidth(max(msg.Width-4, 20))
		return m, nil
	case debounceMsg:
		if msg.seq != m.seq {
			return m, nil
		}
		return m, m.search(msg.seq, strings.TrimSpace(m.input.Value()))
	case searchResultMsg:
		if msg.seq != m.seq {
			return m, nil // an older query answered late
		}
		m.loading, m.problem = false, ""
		if msg.err != nil {
			m.problem = msg.err.Error()
			return m, nil
		}
		m.query, m.results, m.offset = msg.query, msg.items, 0
		// A search is meant to pick a match: put the cursor on the first one.
		if msg.query != "" && len(msg.items) > 0 {
			m.cursor = len(m.opts.Fixed)
		} else {
			m.cursor = min(m.cursor, max(m.total()-1, 0))
		}
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m.finish(ErrInterrupted)
		case "esc":
			return m.finish(ErrCancelled)
		case "enter":
			if m.total() == 0 {
				return m, nil
			}
			if m.cursor < len(m.opts.Fixed) {
				m.result = SearchResult{Fixed: m.cursor, Index: -1, Query: m.query}
			} else {
				m.result = SearchResult{Fixed: -1, Index: m.cursor - len(m.opts.Fixed), Query: m.query}
			}
			return m.finish(nil)
		case "up", "ctrl+p", "ctrl+k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.scroll()
			return m, nil
		case "down", "ctrl+n", "ctrl+j":
			if m.cursor < m.total()-1 {
				m.cursor++
			}
			m.scroll()
			return m, nil
		}
		before := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if m.input.Value() != before {
			m.seq++
			seq := m.seq
			return m, tea.Batch(cmd, tea.Tick(m.opts.Debounce, func(time.Time) tea.Msg { return debounceMsg{seq} }))
		}
		return m, cmd
	}
	var cmds []tea.Cmd
	if m.loading {
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		cmds = append(cmds, cmd)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, tea.Batch(append(cmds, cmd)...)
}

func (m *searchModel) scroll() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+m.opts.MaxRows {
		m.offset = m.cursor - m.opts.MaxRows + 1
	}
}

func (m *searchModel) View() tea.View {
	if m.finished {
		if m.err != nil {
			return tea.NewView("")
		}
		chosen := ""
		if m.result.Fixed >= 0 {
			chosen = m.opts.Fixed[m.result.Fixed]
		} else {
			chosen = m.results[m.result.Index]
		}
		return tea.NewView(doneLine(m.opts.Header+":", chosen) + "\n")
	}
	var b strings.Builder
	if m.opts.Header != "" {
		b.WriteString(styleTitle.Render(m.opts.Header) + "\n")
	}
	b.WriteString(m.input.View())
	if m.loading {
		b.WriteString(" " + m.spin.View())
	}
	b.WriteString("\n")
	for i := m.offset; i < m.total() && i < m.offset+m.opts.MaxRows; i++ {
		line := truncate(m.row(i), 100)
		if i < len(m.opts.Fixed) {
			line = styleDim.Render(line)
		}
		if i == m.cursor {
			b.WriteString(styleCursor.Render("❯ ") + styleSelected.Render(m.row(i)) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}
	label := fmt.Sprintf("  %d results", len(m.results))
	if m.query != "" {
		label = fmt.Sprintf("  %d results for %q", len(m.results), m.query)
	}
	b.WriteString(styleDim.Render(label))
	if m.problem != "" {
		b.WriteString("\n" + styleError.Render("  ✗ "+m.problem))
	}
	return tea.NewView(b.String())
}
