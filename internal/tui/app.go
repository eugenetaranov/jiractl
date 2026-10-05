package tui

import (
	"io"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Screen is one view of the app (the menu, a result list, an actions menu…).
// Screens navigate by returning Push/Pop commands and report results with
// SetStatus/SetError.
type Screen interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Screen, tea.Cmd)
	View(width, height int) string
	// Help is the key help shown at the bottom ("" for the default).
	Help() string
}

// Messages screens use to drive the app.
type (
	pushMsg   struct{ screen Screen }
	popMsg    struct{}
	quitMsg   struct{}
	statusMsg struct {
		text, hint string
		isErr      bool
	}
	headerMsg struct{ text string }
)

// Push shows s on top of the current screen.
func Push(s Screen) tea.Cmd { return func() tea.Msg { return pushMsg{s} } }

// Pop goes back to the previous screen (leaving the app from the first one).
func Pop() tea.Cmd { return func() tea.Msg { return popMsg{} } }

// QuitApp leaves the app.
func QuitApp() tea.Cmd { return func() tea.Msg { return quitMsg{} } }

// SetStatus shows a result in the status bar.
func SetStatus(text string) tea.Cmd {
	return func() tea.Msg { return statusMsg{text: text} }
}

// SetError shows an error, and what to do about it, in the status bar.
func SetError(err error, hint string) tea.Cmd {
	return func() tea.Msg { return statusMsg{text: err.Error(), hint: hint, isErr: true} }
}

// SetHeader sets the line shown above every screen.
func SetHeader(text string) tea.Cmd {
	return func() tea.Msg { return headerMsg{text} }
}

// Suspend leaves the app's screen, runs fn (which may run its own prompts),
// then comes back and delivers done(err).
func Suspend(fn func() error, done func(error) tea.Msg) tea.Cmd {
	return tea.Exec(&funcCommand{fn: fn}, done)
}

type funcCommand struct{ fn func() error }

func (c *funcCommand) Run() error          { return c.fn() }
func (c *funcCommand) SetStdin(io.Reader)  {}
func (c *funcCommand) SetStdout(io.Writer) {}
func (c *funcCommand) SetStderr(io.Writer) {}

// RunApp runs root full-screen until it is left. init runs alongside the
// root screen's Init (e.g. to load the header).
func RunApp(root Screen, init tea.Cmd) error {
	final, err := run(&app{stack: []Screen{root}, init: init, width: 80, height: 24})
	if err != nil {
		return err
	}
	if final.(*app).interrupted {
		return ErrInterrupted
	}
	return nil
}

type app struct {
	stack         []Screen
	init          tea.Cmd
	header        string
	status        statusMsg
	width, height int
	interrupted   bool
}

func (a *app) top() Screen { return a.stack[len(a.stack)-1] }

func (a *app) Init() tea.Cmd { return tea.Batch(a.init, a.top().Init()) }

func (a *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			a.interrupted = true
			return a, tea.Quit
		}
	case pushMsg:
		a.stack = append(a.stack, msg.screen)
		return a, msg.screen.Init()
	case popMsg:
		if len(a.stack) == 1 {
			return a, tea.Quit
		}
		a.stack = a.stack[:len(a.stack)-1]
		return a, nil
	case quitMsg:
		return a, tea.Quit
	case interruptMsg:
		a.interrupted = true
		return a, tea.Quit
	case statusMsg:
		a.status = msg
		return a, nil
	case headerMsg:
		a.header = msg.text
		return a, nil
	}
	s, cmd := a.top().Update(msg)
	a.stack[len(a.stack)-1] = s
	return a, cmd
}

const defaultHelp = "↑↓ move · type to filter · enter choose · esc back · ctrl+c quit"

func (a *app) View() tea.View {
	w := a.width
	header := styleDim.Render(truncate(a.header, w))
	help := a.top().Help()
	if help == "" {
		help = defaultHelp
	}
	var status string
	switch {
	case a.status.isErr:
		status = styleError.Render(truncate("✗ "+a.status.text, w))
		if a.status.hint != "" {
			status += "\n" + styleDim.Render(truncate("  → "+a.status.hint, w))
		}
	case a.status.text != "":
		status = styleDone.Render(truncate("✓ "+a.status.text, w))
	}
	statusLines := max(lipgloss.Height(status), 1)
	bodyHeight := max(a.height-2-statusLines, 3)
	body := a.top().View(w, bodyHeight)
	body = lipgloss.NewStyle().Height(bodyHeight).MaxHeight(bodyHeight).Render(body)

	v := tea.NewView(strings.Join([]string{header, body, status, styleDim.Render(truncate(help, w))}, "\n"))
	v.AltScreen = true
	return v
}

// Interrupt leaves the app as if Ctrl+C was pressed (exit 130).
func Interrupt() tea.Cmd { return func() tea.Msg { return interruptMsg{} } }

type interruptMsg struct{}
