package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type fakeScreen struct {
	name string
	list *List
}

func newFake(name string) *fakeScreen {
	return &fakeScreen{name: name, list: NewList(name, []string{"push", "status", "fail"}, SelectOptions{Header: name})}
}

func (s *fakeScreen) Init() tea.Cmd        { return nil }
func (s *fakeScreen) Help() string         { return "" }
func (s *fakeScreen) View(w, h int) string { return s.list.View(w, h) }
func (s *fakeScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if done, ok := msg.(SelectDoneMsg); ok {
		if done.Err != nil {
			return s, Pop()
		}
		switch done.Index {
		case 0:
			return s, Push(newFake(s.name + ">child"))
		case 1:
			return s, SetStatus("did it")
		case 2:
			return s, SetError(errors.New("broke"), "fix it")
		}
	}
	return s, s.list.Update(msg)
}

// drive applies msg and any resulting commands (one level deep is enough
// for these screens).
func drive(a *app, msg tea.Msg) tea.Cmd {
	_, cmd := a.Update(msg)
	for i := 0; i < 3 && cmd != nil; i++ {
		next := cmd()
		if next == nil {
			return nil
		}
		if _, quit := next.(tea.QuitMsg); quit {
			return tea.Quit
		}
		_, cmd = a.Update(next)
	}
	return cmd
}

func TestAppNavigationAndStatus(t *testing.T) {
	a := &app{stack: []Screen{newFake("root")}, width: 100, height: 20}

	drive(a, press("enter")) // push
	if len(a.stack) != 2 || !strings.Contains(a.View().Content, "root>child") {
		t.Fatalf("push failed: %d screens", len(a.stack))
	}
	drive(a, press("down"))
	drive(a, press("enter")) // status
	if !strings.Contains(a.View().Content, "✓ did it") {
		t.Fatal("status not shown")
	}
	drive(a, press("down"))
	drive(a, press("enter")) // error
	view := a.View().Content
	if !strings.Contains(view, "✗ broke") || !strings.Contains(view, "→ fix it") {
		t.Fatalf("error not shown:\n%s", view)
	}
	drive(a, press("esc")) // pop back to root
	if len(a.stack) != 1 {
		t.Fatalf("pop failed: %d screens", len(a.stack))
	}
	if cmd := drive(a, press("esc")); cmd == nil {
		t.Fatal("esc on the first screen should quit")
	}
}

func TestAppCtrlCInterrupts(t *testing.T) {
	a := &app{stack: []Screen{newFake("root")}, width: 80, height: 20}
	a.Update(press("ctrl+c"))
	if !a.interrupted {
		t.Fatal("ctrl+c did not interrupt")
	}
}
