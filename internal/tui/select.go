package tui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
)

// SelectOptions configures Select.
type SelectOptions struct {
	// Header is shown above the filter, e.g. "Select issue type".
	Header string
	// Preview, when set, renders details for items[i] in a pane beside the
	// list (below it in terminals narrower than 100 columns).
	Preview func(i, width, height int) string
	// MaxRows caps the list height (default 12).
	MaxRows int
	// Check, when set, validates the chosen item asynchronously; a failure
	// is shown under the list and another item can be picked. After
	// MaxAttempts failures the error is returned.
	Check       func(i int) error
	CheckLabel  string
	MaxAttempts int
}

// Select shows items in an inline list filtered as the user types, and
// returns the chosen index. Space-separated words must all match.
func Select(items []string, opts SelectOptions) (int, error) {
	m := newSelectModel(items, opts)
	final, err := run(m)
	if err != nil {
		return 0, err
	}
	sm := final.(*selectModel)
	if sm.err != nil {
		return 0, sm.err
	}
	return sm.chosen, nil
}

type selectModel struct {
	// embedded lists live inside the app: choosing or Esc sends a
	// SelectDoneMsg instead of ending the program, and the list stays usable.
	embedded bool
	id       string
	// fill uses the whole height given to View instead of a bounded height.
	fill     bool
	items    []string
	opts     SelectOptions
	filter   textinput.Model
	check    checker
	problem  string
	matches  []int // indices into items, in display order
	cursor   int   // position in matches
	offset   int   // first visible match
	width    int
	height   int
	chosen   int
	err      error
	finished bool
}

func newSelectModel(items []string, opts SelectOptions) *selectModel {
	if opts.MaxRows <= 0 {
		opts.MaxRows = 12
	}
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "type to filter"
	ti.SetWidth(40)
	ti.Focus()
	m := &selectModel{items: items, opts: opts, filter: ti, width: 80, height: 24, chosen: -1,
		check: newChecker(opts.CheckLabel, opts.MaxAttempts)}
	m.refilter()
	return m
}

func (m *selectModel) Init() tea.Cmd { return textinput.Blink }

// matchItems returns the indices of items matching every word of query,
// best matches first; an empty query keeps the original order.
func matchItems(items []string, query string) []int {
	words := strings.Fields(query)
	if len(words) == 0 {
		all := make([]int, len(items))
		for i := range items {
			all[i] = i
		}
		return all
	}
	score := map[int]int{}
	for wi, w := range words {
		found := map[int]int{}
		for _, match := range fuzzy.Find(w, items) {
			found[match.Index] = match.Score
		}
		next := map[int]int{}
		for idx, s := range found {
			if wi == 0 {
				next[idx] = s
			} else if prev, ok := score[idx]; ok {
				next[idx] = prev + s
			}
		}
		score = next
	}
	out := make([]int, 0, len(score))
	for idx := range score {
		out = append(out, idx)
	}
	sort.Slice(out, func(a, b int) bool {
		if score[out[a]] != score[out[b]] {
			return score[out[a]] > score[out[b]]
		}
		return out[a] < out[b]
	})
	return out
}

func (m *selectModel) refilter() {
	m.matches = matchItems(m.items, m.filter.Value())
	m.cursor, m.offset = 0, 0
}

// SelectDoneMsg reports a choice (or Esc as ErrCancelled) from an embedded
// list.
type SelectDoneMsg struct {
	ID    string
	Index int
	Err   error
}

// end finishes the list: a standalone list quits its program, an embedded
// one reports to its screen and stays ready for the next choice.
func (m *selectModel) end(err error) tea.Cmd {
	if !m.embedded {
		m.err, m.finished = err, true
		return tea.Quit
	}
	msg := SelectDoneMsg{ID: m.id, Index: m.chosen, Err: err}
	return func() tea.Msg { return msg }
}

func (m *selectModel) rows() int {
	if m.fill {
		overhead := 2 // filter + counter
		if m.opts.Header != "" {
			overhead++
		}
		if m.opts.Preview != nil && m.width < 100 {
			return max((m.height-overhead)/2, 1)
		}
		return max(min(m.height-overhead, len(m.items)), 1)
	}
	rows := min(m.opts.MaxRows, len(m.items))
	if limit := m.height/2 - 3; limit > 3 && rows > limit {
		rows = limit
	}
	return max(rows, 1)
}

func (m *selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if res, ok := msg.(checkResultMsg); ok {
		current, giveUp := m.check.result(res)
		switch {
		case !current:
			return m, nil
		case res.err == nil:
			return m, m.end(nil)
		case giveUp:
			return m, m.end(res.err)
		}
		m.problem = res.err.Error()
		m.chosen = -1
		return m, nil
	}
	if m.check.running {
		if keyIs(msg, "ctrl+c") {
			return m, m.end(ErrInterrupted)
		}
		if keyIs(msg, "esc") {
			return m, m.end(ErrCancelled)
		}
		return m, m.check.update(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.filter.SetWidth(max(msg.Width-4, 20))
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, m.end(ErrInterrupted)
		case "esc":
			return m, m.end(ErrCancelled)
		case "enter":
			if len(m.matches) == 0 {
				return m, nil
			}
			m.chosen = m.matches[m.cursor]
			if m.opts.Check != nil {
				m.problem = ""
				chosen := m.chosen
				return m, m.check.start(func() error { return m.opts.Check(chosen) })
			}
			return m, m.end(nil)
		case "up", "ctrl+p", "ctrl+k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n", "ctrl+j":
			if m.cursor < len(m.matches)-1 {
				m.cursor++
			}
		case "pgup":
			m.cursor = max(m.cursor-m.rows(), 0)
		case "pgdown":
			m.cursor = min(m.cursor+m.rows(), max(len(m.matches)-1, 0))
		default:
			before := m.filter.Value()
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			if m.filter.Value() != before {
				m.refilter()
			}
			return m, cmd
		}
		rows := m.rows()
		if m.cursor < m.offset {
			m.offset = m.cursor
		} else if m.cursor >= m.offset+rows {
			m.offset = m.cursor - rows + 1
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	return m, cmd
}

func (m *selectModel) listView(width int) string {
	var b strings.Builder
	rows := m.rows()
	for i := m.offset; i < len(m.matches) && i < m.offset+rows; i++ {
		line := truncate(m.items[m.matches[i]], width-2)
		if i == m.cursor {
			b.WriteString(styleCursor.Render("❯ ") + styleSelected.Render(line))
		} else {
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
	}
	if len(m.matches) == 0 {
		b.WriteString(styleDim.Render("  no matches") + "\n")
	}
	b.WriteString(styleDim.Render(fmt.Sprintf("  %d/%d", len(m.matches), len(m.items))))
	switch {
	case m.check.running:
		b.WriteString("\n  " + m.check.view())
	case m.problem != "":
		b.WriteString("\n" + styleError.Render("  ✗ "+m.problem))
	}
	return b.String()
}

func (m *selectModel) View() tea.View {
	return tea.NewView(m.render())
}

func (m *selectModel) render() string {
	if m.finished {
		switch {
		case m.err == nil && m.chosen >= 0 && m.opts.Header != "":
			return doneLine(m.opts.Header+":", m.items[m.chosen]) + "\n"
		case m.err != nil && !errorsIsCancel(m.err):
			return failLine(m.opts.Header+":", m.err.Error()) + "\n"
		}
		return ""
	}

	var b strings.Builder
	if m.opts.Header != "" {
		b.WriteString(styleTitle.Render(m.opts.Header) + "\n")
	}
	b.WriteString(m.filter.View() + "\n")

	if m.opts.Preview == nil || len(m.matches) == 0 {
		b.WriteString(m.listView(m.width))
		return b.String()
	}

	current := m.matches[m.cursor]
	if m.width >= 100 {
		listWidth := m.width * 45 / 100
		list := lipgloss.NewStyle().Width(listWidth).Render(m.listView(listWidth))
		paneHeight := max(m.rows()+1, 8)
		pane := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			PaddingLeft(1).
			Width(m.width - listWidth - 1).
			Height(paneHeight).
			MaxHeight(paneHeight).
			Render(m.opts.Preview(current, m.width-listWidth-3, paneHeight))
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, list, pane))
		return b.String()
	}

	b.WriteString(m.listView(m.width) + "\n")
	paneHeight := 8
	if m.fill {
		paneHeight = max(m.height-m.rows()-5, 3)
	}
	pane := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		Width(m.width).
		MaxHeight(paneHeight + 1).
		Render(m.opts.Preview(current, m.width, paneHeight))
	b.WriteString(pane)
	return b.String()
}

// truncate cuts s to width display columns.
func truncate(s string, width int) string {
	if width <= 1 || lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// List is a Select that lives inside the app (see App): it reports choices
// with SelectDoneMsg and fills the space it is given.
type List struct {
	m *selectModel
}

// NewList creates an embedded list; id identifies it in SelectDoneMsg.
func NewList(id string, items []string, opts SelectOptions) *List {
	m := newSelectModel(items, opts)
	m.embedded, m.fill, m.id = true, true, id
	return &List{m: m}
}

// Init starts the cursor blink.
func (l *List) Init() tea.Cmd { return l.m.Init() }

// Update handles a message.
func (l *List) Update(msg tea.Msg) tea.Cmd {
	_, cmd := l.m.Update(msg)
	return cmd
}

// View renders the list into width x height.
func (l *List) View(width, height int) string {
	l.m.width, l.m.height = width, height
	l.m.filter.SetWidth(max(width-4, 20))
	return l.m.render()
}

// SetItems replaces the items, keeping the filter and, when possible, the
// cursor position.
func (l *List) SetItems(items []string) {
	cursor := l.m.cursor
	l.m.items = items
	l.m.matches = matchItems(items, l.m.filter.Value())
	l.m.cursor = min(cursor, max(len(l.m.matches)-1, 0))
}

// Current returns the item index under the cursor, or -1.
func (l *List) Current() int {
	if len(l.m.matches) == 0 {
		return -1
	}
	return l.m.matches[l.m.cursor]
}
