package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/charmbracelet/x/exp/teatest/v2"
)

// press builds a key press: "enter", "esc", "ctrl+c", "up", "down", or text.
func press(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "ctrl+c", "ctrl+d", "ctrl+e":
		return tea.KeyPressMsg{Code: rune(s[5]), Mod: tea.ModCtrl}
	}
	r := []rune(s)
	return tea.KeyPressMsg{Code: r[0], Text: s}
}

// feed sends keys to a model, typing multi-character strings one rune at a
// time.
func feed(m tea.Model, keys ...string) tea.Model {
	for _, k := range keys {
		if len([]rune(k)) > 1 && !strings.Contains("enter esc up down ctrl+c ctrl+d ctrl+e", k) {
			for _, r := range k {
				m, _ = m.Update(press(string(r)))
			}
			continue
		}
		m, _ = m.Update(press(k))
	}
	return m
}

func TestKeyHelper(t *testing.T) {
	for _, k := range []string{"enter", "esc", "ctrl+c", "ctrl+d", "up", "a"} {
		if got := press(k).String(); got != k {
			t.Errorf("press(%q).String() = %q", k, got)
		}
	}
}

var menu = []string{"Create new issue", "Run query", "Change default epic", "Configure", "Exit"}

func TestMatchItemsAllWords(t *testing.T) {
	got := matchItems(menu, "change default")
	if len(got) != 1 || menu[got[0]] != "Change default epic" {
		t.Fatalf("got %v", got)
	}
	if got := matchItems(menu, ""); len(got) != len(menu) || got[0] != 0 || got[4] != 4 {
		t.Fatalf("empty query reordered: %v", got)
	}
	if got := matchItems(menu, "zzz"); len(got) != 0 {
		t.Fatalf("expected no matches, got %v", got)
	}
}

func TestSelectKeys(t *testing.T) {
	m := feed(newSelectModel(menu, SelectOptions{}), "down", "down", "up", "enter").(*selectModel)
	if m.chosen != 1 || m.err != nil {
		t.Fatalf("chosen=%d err=%v", m.chosen, m.err)
	}
	m = feed(newSelectModel(menu, SelectOptions{}), "conf", "enter").(*selectModel)
	if menu[m.chosen] != "Configure" {
		t.Fatalf("filtered choice %q", menu[m.chosen])
	}
	m = feed(newSelectModel(menu, SelectOptions{}), "esc").(*selectModel)
	if !IsEsc(m.err) {
		t.Fatalf("esc: %v", m.err)
	}
	m = feed(newSelectModel(menu, SelectOptions{}), "ctrl+c").(*selectModel)
	if !errors.Is(m.err, ErrInterrupted) || IsEsc(m.err) || !errors.Is(m.err, ErrCancelled) {
		t.Fatalf("ctrl+c: %v", m.err)
	}
	m = feed(newSelectModel(menu, SelectOptions{}), "zzz", "enter").(*selectModel)
	if m.finished {
		t.Fatal("enter with no matches must not finish")
	}
}

func TestInputRequiredAndDefault(t *testing.T) {
	m := feed(newInputModel("Summary", InputOptions{Required: true}, false), "enter").(*inputModel)
	if m.finished || m.problem == "" {
		t.Fatal("empty required input accepted")
	}
	m = feed(m, "Fix login", "enter").(*inputModel)
	if !m.finished || m.value != "Fix login" {
		t.Fatalf("value %q", m.value)
	}
	m = feed(newInputModel("Server", InputOptions{Default: "https://x"}, false), "enter").(*inputModel)
	if m.value != "https://x" {
		t.Fatalf("default %q", m.value)
	}
	m = feed(newInputModel("Token", InputOptions{}, true), "s3cret", "enter").(*inputModel)
	if m.value != "s3cret" || strings.Contains(m.View().Content, "s3cret") {
		t.Fatalf("secret leaked or wrong: %q", m.View().Content)
	}
}

func TestChoice(t *testing.T) {
	for keys, want := range map[string]byte{"e": 'e', "enter": 'y', "N": 'n'} {
		m := feed(&choiceModel{label: "Create?", valid: "yedn"}, keys).(*choiceModel)
		if m.chosen != want {
			t.Errorf("%s: got %q", keys, m.chosen)
		}
	}
	m := feed(&choiceModel{label: "Create?", valid: "yedn"}, "x").(*choiceModel)
	if m.finished {
		t.Fatal("invalid key accepted")
	}
}

func TestTextarea(t *testing.T) {
	m := feed(newTextareaModel("Description", ""), "First", "enter", "enter", "Second", "enter", ".", "ctrl+d").(*textareaModel)
	if m.value != "First\n\nSecond\n." {
		t.Fatalf("value %q", m.value)
	}
	m = feed(newTextareaModel("Description", "keep"), "esc").(*textareaModel)
	if !IsEsc(m.err) {
		t.Fatalf("esc: %v", m.err)
	}
}

func TestSelectProgram(t *testing.T) {
	tm := teatest.NewTestModel(t, newSelectModel(menu, SelectOptions{Header: "Select action"}), teatest.WithInitialTermSize(80, 24))
	tm.Type("query")
	tm.Send(press("enter"))
	final := tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second)).(*selectModel)
	if menu[final.chosen] != "Run query" {
		t.Fatalf("chosen %q", menu[final.chosen])
	}
}

func previewOf(i, w, h int) string {
	return "Preview of item " + string(rune('A'+i)) + "\nsecond line"
}

func TestSelectViewGolden(t *testing.T) {
	for name, width := range map[string]int{"side_by_side": 120, "stacked": 80} {
		t.Run(name, func(t *testing.T) {
			m := newSelectModel(menu, SelectOptions{Header: "Pick one", Preview: previewOf})
			m.width, m.height = width, 30
			golden.RequireEqual(t, []byte(m.View().Content))
		})
	}
}

// runCheck feeds the check result for a model that just started a check.
func runCheck(t *testing.T, m tea.Model, cmd tea.Cmd) tea.Model {
	t.Helper()
	for cmd != nil {
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				if res, ok := c().(checkResultMsg); ok {
					m, _ = m.Update(res)
					return m
				}
			}
			return m
		}
		m, cmd = m.Update(msg)
	}
	return m
}

func TestInputCheckRetriesThenGivesUp(t *testing.T) {
	calls := 0
	check := func(v string) error {
		calls++
		if v == "good" {
			return nil
		}
		return errors.New("rejected")
	}
	m := tea.Model(newInputModel("Server", InputOptions{Check: check, MaxAttempts: 2}, false))
	m = feed(m, "bad")
	m, cmd := m.Update(press("enter"))
	m = runCheck(t, m, cmd)
	im := m.(*inputModel)
	if im.finished || im.problem != "rejected" || !strings.Contains(im.View().Content, "✗ rejected") {
		t.Fatalf("after first failure: finished=%v problem=%q", im.finished, im.problem)
	}
	// Correct the value: the check passes and the field finishes.
	im = m.(*inputModel)
	im.input.SetValue("good")
	m, cmd = m.Update(press("enter"))
	m = runCheck(t, m, cmd)
	if im := m.(*inputModel); !im.finished || im.err != nil || im.value != "good" {
		t.Fatalf("good value: %+v", im.err)
	}

	m = tea.Model(newInputModel("Server", InputOptions{Check: check, MaxAttempts: 2}, false))
	for i := 0; i < 2; i++ {
		m = feed(m, "x")
		var cmd tea.Cmd
		m, cmd = m.Update(press("enter"))
		m = runCheck(t, m, cmd)
	}
	if im := m.(*inputModel); !im.finished || im.err == nil || im.err.Error() != "rejected" {
		t.Fatalf("expected give-up error, got %v", im.err)
	}
}

func TestSelectCheckStaysOnFailure(t *testing.T) {
	m := tea.Model(newSelectModel(menu, SelectOptions{Check: func(i int) error {
		if i == 0 {
			return errors.New("not allowed")
		}
		return nil
	}}))
	m, cmd := m.Update(press("enter"))
	m = runCheck(t, m, cmd)
	if sm := m.(*selectModel); sm.finished || sm.problem != "not allowed" {
		t.Fatalf("finished=%v problem=%q", sm.finished, sm.problem)
	}
	m = feed(m, "down")
	m, cmd = m.Update(press("enter"))
	m = runCheck(t, m, cmd)
	if sm := m.(*selectModel); !sm.finished || sm.chosen != 1 {
		t.Fatalf("finished=%v chosen=%d", sm.finished, sm.chosen)
	}
}

func TestReview(t *testing.T) {
	rows := []ReviewRow{{"Type", "Task"}, {"Summary", "Fix login"}, {"Description", "a\nb\nc\nd\ne\nf\ng\nh"}}
	m := feed(&reviewModel{title: "Creating issue in OPS:", rows: rows, choice: choiceModel{label: "Create?", valid: "yedn"}}, "e").(*reviewModel)
	if m.choice.chosen != 'e' || m.View().Content != "" {
		t.Fatalf("edit: chosen=%q view=%q", m.choice.chosen, m.View().Content)
	}
	m = &reviewModel{title: "Creating issue in OPS:", rows: rows, choice: choiceModel{label: "Create?", valid: "yedn"}}
	view := m.View().Content
	for _, want := range []string{"Summary:", "Fix login", "… 2 more lines", "[Y/e/d/n]"} {
		if !strings.Contains(view, want) {
			t.Errorf("view lacks %q", want)
		}
	}
	m = feed(m, "enter").(*reviewModel)
	if m.choice.chosen != 'y' || !strings.Contains(m.View().Content, "Fix login") {
		t.Fatal("table should stay after confirming")
	}
}

func TestSearchDropsStaleResults(t *testing.T) {
	m := newSearchModel(SearchOptions{Fixed: []string{"Skip"}, Search: func(q string) ([]string, error) { return nil, nil }})
	m.seq = 2
	m.Update(searchResultMsg{seq: 1, query: "dev", items: []string{"OPS-1 dev"}})
	if len(m.results) != 0 {
		t.Fatal("stale results applied")
	}
	m.Update(searchResultMsg{seq: 2, query: "devops", items: []string{"OPS-40 DevOps k8s"}})
	if len(m.results) != 1 || m.cursor != 1 {
		t.Fatalf("results=%v cursor=%d", m.results, m.cursor)
	}
	m.Update(press("enter"))
	if m.result.Index != 0 || m.result.Query != "devops" || m.result.Fixed != -1 {
		t.Fatalf("result %+v", m.result)
	}
}

func TestSearchDebouncesTyping(t *testing.T) {
	m := newSearchModel(SearchOptions{Search: func(q string) ([]string, error) { return nil, nil }})
	m.Update(press("a"))
	m.Update(press("b"))
	if m.seq != 2 {
		t.Fatalf("seq=%d", m.seq)
	}
	// The first keystroke's timer fires late: no search for it.
	if _, cmd := m.Update(debounceMsg{seq: 1}); cmd != nil || m.loading {
		t.Fatal("stale debounce started a search")
	}
	if _, cmd := m.Update(debounceMsg{seq: 2}); cmd == nil || !m.loading {
		t.Fatal("current debounce did not search")
	}
}

func TestSearchFixedRowFirst(t *testing.T) {
	m := newSearchModel(SearchOptions{Fixed: []string{"Skip"}, Search: func(q string) ([]string, error) { return nil, nil }})
	m.Update(searchResultMsg{seq: 0, query: "", items: []string{"OPS-1"}})
	if m.cursor != 0 {
		t.Fatalf("cursor %d, want the fixed row", m.cursor)
	}
	m.Update(press("enter"))
	if m.result.Fixed != 0 {
		t.Fatalf("result %+v", m.result)
	}
}

func TestViewsGolden(t *testing.T) {
	search := newSearchModel(SearchOptions{Header: "Search epics", Fixed: []string{"Skip: create without epic"}, Search: func(string) ([]string, error) { return nil, nil }})
	search.Update(searchResultMsg{seq: 0, query: "", items: []string{"OPS-40 - DevOps k8s cluster upgrade", "OPS-41 - Billing revamp [done]"}})

	inputWithError := newInputModel("Jira Server URL", InputOptions{Default: "https://acme.atlassian.net"}, false)
	inputWithError.problem = "cannot reach Jira at https://acme.atlassian.net: no such host"

	app := &app{stack: []Screen{newFake("Select action")}, width: 80, height: 12,
		header: "Default epic: OPS-40 DevOps k8s cluster upgrade",
		status: statusMsg{text: "Jira rejected your credentials (401)", hint: "Create a new API token", isErr: true}}

	views := map[string]string{
		"input":       newInputModel("Summary", InputOptions{Required: true}, false).View().Content,
		"input_error": inputWithError.View().Content,
		"secret":      newInputModel("API Token", InputOptions{Hint: "(leave empty to keep the stored one)"}, true).View().Content,
		"choice":      (&choiceModel{label: "Create?", valid: "yedn"}).View().Content,
		"textarea":    newTextareaModel("Description", "First\n\nSecond").View().Content,
		"review":      (&reviewModel{title: "Creating issue in OPS:", rows: []ReviewRow{{"Type", "Task"}, {"Summary", "Fix login"}, {"Epic", "OPS-40 - DevOps k8s (default)"}}, choice: choiceModel{label: "Create?", valid: "yedn"}}).View().Content,
		"search":      search.View().Content,
		"app_menu":    app.View().Content,
	}
	for name, view := range views {
		t.Run(name, func(t *testing.T) {
			golden.RequireEqual(t, []byte(view))
		})
	}
}
