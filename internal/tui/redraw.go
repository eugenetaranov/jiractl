package tui

import (
	"go/token"
	"os"
	"reflect"

	tea "charm.land/bubbletea/v2"
)

// fullRedraw is a test hook (JIRACTL_E2E_REDRAW=1). Bubble Tea writes only
// the cells that changed, so screen text reaches the terminal in fragments;
// the end-to-end tests read the raw stream and need whole lines. With the
// hook every update is followed by a full repaint.
var fullRedraw = os.Getenv("JIRACTL_E2E_REDRAW") == "1"

const teaPkg = "charm.land/bubbletea/v2"

type redrawModel struct{ tea.Model }

func (r redrawModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := r.Model.Update(msg)
	// Bubble Tea's unexported internal messages (including the one
	// ClearScreen sends) must not trigger another repaint, or it would
	// never stop.
	if t := reflect.TypeOf(msg); t != nil && t.PkgPath() == teaPkg && !token.IsExported(t.Name()) {
		return redrawModel{m}, cmd
	}
	return redrawModel{m}, tea.Batch(cmd, tea.ClearScreen)
}
