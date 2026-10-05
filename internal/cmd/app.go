package cmd

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	jiralib "github.com/andygrunwald/go-jira"
	"github.com/atotto/clipboard"
	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/tui"
)

// The app shell: the main menu and query browsing in one full-screen
// program. Create, configure and "Change default epic" keep their own
// prompts; the app steps aside while they run (tui.Suspend) and shows their
// result in the status bar.

// actionDoneMsg reports a finished action to the status bar.
type actionDoneMsg struct {
	status string
	err    error
	// refreshHeader reloads the default epic / login line.
	refreshHeader bool
}

// report turns an action result into status bar commands.
func report(msg actionDoneMsg, cfgChanged func() tea.Cmd) tea.Cmd {
	var cmds []tea.Cmd
	switch {
	case errors.Is(msg.err, tui.ErrInterrupted):
		return tui.Interrupt()
	case tui.IsEsc(msg.err):
		cmds = append(cmds, tui.SetStatus("Cancelled"))
	case msg.err != nil:
		cmds = append(cmds, tui.SetError(msg.err, errorHint(msg.err)))
	case msg.status != "":
		cmds = append(cmds, tui.SetStatus(msg.status))
	}
	if msg.refreshHeader && cfgChanged != nil {
		cmds = append(cmds, cfgChanged())
	}
	return tea.Batch(cmds...)
}

// headerCmd loads the line above every screen: the default epic, and a
// warning when Jira rejects the stored token.
func headerCmd() tea.Cmd {
	return func() tea.Msg {
		line := ""
		if cfg, err := config.Load(); err == nil {
			client, _ := jira.NewClient(cfg)
			line = defaultEpicLine(cfg, client)
		}
		if warning := checkLogin(); warning != "" {
			line = "⚠ " + warning + "  │  " + line
		}
		return tui.SetHeader(line)()
	}
}

// --- main menu

type menuScreen struct {
	list *tui.List
}

func newMenuScreen() *menuScreen {
	return &menuScreen{list: tui.NewList("menu", menuItems, tui.SelectOptions{Header: "Select action"})}
}

func (s *menuScreen) Init() tea.Cmd { return s.list.Init() }
func (s *menuScreen) Help() string  { return "" }

func (s *menuScreen) View(w, h int) string { return s.list.View(w, h) }

func (s *menuScreen) Update(msg tea.Msg) (tui.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case actionDoneMsg:
		return s, report(msg, headerCmd)
	case tui.SelectDoneMsg:
		if msg.Err != nil {
			return s, tui.QuitApp()
		}
		switch menuItems[msg.Index] {
		case "Create new issue":
			return s, suspend(func() (string, error) {
				issue, _, err := createIssueFlow()
				if err != nil {
					return "", err
				}
				cfg, _ := config.Load()
				return fmt.Sprintf("Created %s  %s/browse/%s", issue.Key, cfg.Server, issue.Key), nil
			}, false)
		case "Run query":
			return s, openQueryPicker()
		case "Change default epic":
			return s, suspend(changeDefaultEpic, true)
		case "Configure":
			return s, suspend(func() (string, error) {
				return "Configuration saved", runConfigure(configureCmd, nil)
			}, true)
		case "Exit":
			return s, tui.QuitApp()
		}
		return s, nil
	}
	return s, s.list.Update(msg)
}

// suspend runs fn outside the app and reports its result.
func suspend(fn func() (string, error), refreshHeader bool) tea.Cmd {
	var status string
	return tui.Suspend(func() error {
		var err error
		status, err = fn()
		return err
	}, func(err error) tea.Msg {
		if err != nil {
			status = ""
		}
		return actionDoneMsg{status: status, err: err, refreshHeader: refreshHeader}
	})
}

// --- saved query picker

type queryPickScreen struct {
	cfg     *config.Config
	list    *tui.List
	loading string
}

func openQueryPicker() tea.Cmd {
	cfg, err := loadConfig()
	if err != nil {
		return tui.SetError(err, errorHint(err))
	}
	if len(cfg.Queries) == 0 {
		return tui.SetError(errors.New("no queries configured"), "add [[queries]] to ~/"+config.ConfigFileName+" or run Configure to add starter queries")
	}
	return tui.Push(&queryPickScreen{cfg: cfg, list: tui.NewList("queries", cfg.QueryNames(), tui.SelectOptions{Header: "Select query"})})
}

type queryLoadedMsg struct {
	title  string
	issues []jiralib.Issue
	client *jira.Client
	err    error
}

func (s *queryPickScreen) Init() tea.Cmd { return s.list.Init() }
func (s *queryPickScreen) Help() string  { return "" }

func (s *queryPickScreen) View(w, h int) string {
	if s.loading != "" {
		return "Running query " + s.loading + "…"
	}
	return s.list.View(w, h)
}

func (s *queryPickScreen) Update(msg tea.Msg) (tui.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.SelectDoneMsg:
		if msg.Err != nil {
			return s, tui.Pop()
		}
		q := s.cfg.Queries[msg.Index]
		s.loading = q.Name
		return s, loadQuery(s.cfg, q.Name, s.cfg.ExpandJQL(q.JQL), q.Limit)
	case queryLoadedMsg:
		s.loading = ""
		switch {
		case msg.err != nil:
			return s, tui.SetError(msg.err, errorHint(msg.err))
		case len(msg.issues) == 0:
			return s, tui.SetStatus(fmt.Sprintf("No issues found for %q", msg.title))
		}
		return s, tea.Batch(
			tui.SetStatus(fmt.Sprintf("%s: %d issues", msg.title, len(msg.issues))),
			tui.Push(newResultsScreen(s.cfg, msg.client, msg.title, msg.issues)),
		)
	}
	return s, s.list.Update(msg)
}

// loadQuery runs a search in the background.
func loadQuery(cfg *config.Config, title, jql string, limit int) tea.Cmd {
	return func() tea.Msg {
		client, err := jira.NewClient(cfg)
		if err != nil {
			return queryLoadedMsg{title: title, err: err}
		}
		if limit <= 0 {
			limit = 50
		}
		issues, err := client.SearchIssues(jql, limit)
		if err != nil {
			return queryLoadedMsg{title: title, err: fmt.Errorf("query %q failed: %w", title, err)}
		}
		jira.SortForDisplay(issues)
		return queryLoadedMsg{title: title, issues: issues, client: client}
	}
}

// --- results with preview

type resultsScreen struct {
	cfg    *config.Config
	client *jira.Client
	title  string
	issues []jiralib.Issue
	list   *tui.List
}

func newResultsScreen(cfg *config.Config, client *jira.Client, title string, issues []jiralib.Issue) *resultsScreen {
	s := &resultsScreen{cfg: cfg, client: client, title: title, issues: issues}
	s.list = tui.NewList("results", s.rows(), tui.SelectOptions{
		Header:  fmt.Sprintf("%s (%d found)", title, len(issues)),
		Preview: func(i, w, h int) string { return issuePreview(s.issues[i], w, h) },
	})
	return s
}

func (s *resultsScreen) rows() []string {
	rows := make([]string, len(s.issues))
	for i := range s.issues {
		rows[i] = issueRow(s.issues[i])
	}
	return rows
}

// replace updates one issue after an action changed it.
func (s *resultsScreen) replace(issue jiralib.Issue) {
	for i := range s.issues {
		if s.issues[i].Key == issue.Key {
			s.issues[i] = issue
		}
	}
	s.list.SetItems(s.rows())
}

func (s *resultsScreen) Init() tea.Cmd { return s.list.Init() }
func (s *resultsScreen) Help() string {
	return "↑↓ move · type to filter · enter actions · esc back · ctrl+c quit"
}
func (s *resultsScreen) View(w, h int) string { return s.list.View(w, h) }

func (s *resultsScreen) Update(msg tea.Msg) (tui.Screen, tea.Cmd) {
	if done, ok := msg.(tui.SelectDoneMsg); ok {
		if done.Err != nil {
			return s, tui.Pop()
		}
		return s, tui.Push(newActionsScreen(s, s.issues[done.Index]))
	}
	return s, s.list.Update(msg)
}

// --- actions on one issue

type actionsScreen struct {
	results *resultsScreen
	issue   jiralib.Issue
	list    *tui.List
}

func newActionsScreen(results *resultsScreen, issue jiralib.Issue) *actionsScreen {
	return &actionsScreen{results: results, issue: issue,
		list: tui.NewList("actions", issueActions, tui.SelectOptions{Header: issue.Key + ": " + fieldSummary(issue)})}
}

func (s *actionsScreen) Init() tea.Cmd        { return s.list.Init() }
func (s *actionsScreen) Help() string         { return "" }
func (s *actionsScreen) View(w, h int) string { return s.list.View(w, h) }

// issueChangedMsg carries a refreshed issue after an action.
type issueChangedMsg struct {
	status string
	issue  *jiralib.Issue
	err    error
}

type transitionsLoadedMsg struct {
	transitions []jiralib.Transition
	err         error
}

// refreshAfter runs fn in the background, then reloads the issue.
func (s *actionsScreen) refreshAfter(status string, fn func() error) tea.Cmd {
	client, key := s.results.client, s.issue.Key
	return func() tea.Msg {
		if err := fn(); err != nil {
			return issueChangedMsg{err: err}
		}
		fresh, err := client.SearchIssues("key = "+key, 1)
		if err != nil || len(fresh) != 1 {
			return issueChangedMsg{status: status}
		}
		return issueChangedMsg{status: status, issue: &fresh[0]}
	}
}

func (s *actionsScreen) Update(msg tea.Msg) (tui.Screen, tea.Cmd) {
	client, key := s.results.client, s.issue.Key
	switch msg := msg.(type) {
	case issueChangedMsg:
		if msg.err != nil {
			return s, tui.SetError(msg.err, errorHint(msg.err))
		}
		if msg.issue != nil {
			s.issue = *msg.issue
			s.results.replace(*msg.issue)
		}
		return s, tui.SetStatus(msg.status)
	case transitionsLoadedMsg:
		if msg.err != nil {
			return s, tui.SetError(msg.err, errorHint(msg.err))
		}
		if len(msg.transitions) == 0 {
			return s, tui.SetStatus("No transitions available for " + key)
		}
		return s, tui.Push(newTransitionScreen(s, msg.transitions))
	case actionDoneMsg:
		if msg.err == nil && msg.status != "" {
			return s, s.refreshAfter(msg.status, func() error { return nil })
		}
		return s, report(msg, nil)
	case tui.SelectDoneMsg:
		if msg.Err != nil {
			return s, tui.Pop()
		}
		url := fmt.Sprintf("%s/browse/%s", s.results.cfg.Server, key)
		switch issueActions[msg.Index] {
		case "Open in browser":
			if err := openBrowser(url); err != nil {
				return s, tui.SetError(fmt.Errorf("could not open a browser: %w", err), url)
			}
			return s, tui.SetStatus("Opened " + url)
		case "Copy key":
			if err := clipboard.WriteAll(key); err != nil {
				return s, tui.SetError(fmt.Errorf("clipboard unavailable: %w", err), "key: "+key)
			}
			return s, tui.SetStatus("Copied " + key)
		case "Transition":
			return s, func() tea.Msg {
				t, err := client.GetTransitions(key)
				return transitionsLoadedMsg{t, err}
			}
		case "Assign to me":
			return s, s.refreshAfter(key+" assigned to you", func() error { return client.AssignToMe(key) })
		case "Comment":
			return s, suspend(func() (string, error) {
				text, err := promptMultilineText("Comment on " + key)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(text) == "" {
					return "", nil
				}
				if err := client.AddComment(key, text); err != nil {
					return "", err
				}
				return "Comment added to " + key, nil
			}, false)
		case "Back":
			return s, tui.Pop()
		}
		return s, nil
	}
	return s, s.list.Update(msg)
}

// --- transition picker

type transitionScreen struct {
	actions     *actionsScreen
	transitions []jiralib.Transition
	names       []string
	list        *tui.List
}

func newTransitionScreen(actions *actionsScreen, transitions []jiralib.Transition) *transitionScreen {
	names := make([]string, len(transitions))
	for i, t := range transitions {
		names[i] = t.Name
		if t.To.Name != "" && t.To.Name != t.Name {
			names[i] += " → " + t.To.Name
		}
	}
	return &transitionScreen{actions: actions, transitions: transitions, names: names,
		list: tui.NewList("transitions", names, tui.SelectOptions{Header: "Transition " + actions.issue.Key})}
}

func (s *transitionScreen) Init() tea.Cmd        { return s.list.Init() }
func (s *transitionScreen) Help() string         { return "" }
func (s *transitionScreen) View(w, h int) string { return s.list.View(w, h) }

func (s *transitionScreen) Update(msg tea.Msg) (tui.Screen, tea.Cmd) {
	if done, ok := msg.(tui.SelectDoneMsg); ok {
		if done.Err != nil {
			return s, tui.Pop()
		}
		client, key := s.actions.results.client, s.actions.issue.Key
		t := s.transitions[done.Index]
		// Back to the actions screen, which applies the result.
		return s, tea.Sequence(tui.Pop(), s.actions.refreshAfter(key+": "+s.names[done.Index], func() error {
			return client.DoTransition(key, t.ID)
		}))
	}
	return s, s.list.Update(msg)
}
