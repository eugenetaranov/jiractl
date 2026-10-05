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
