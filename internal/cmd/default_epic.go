package cmd

import (
	"fmt"
	"sync"
	"time"

	jiralib "github.com/andygrunwald/go-jira"
	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/textutil"
	"github.com/eugenetaranov/jiractl/internal/tui"
)

// epicLookups caches epic checks for the process, so the menu header and
// create share one request per epic.
var (
	epicLookupsMu sync.Mutex
	epicLookups   = map[string]*pending[epicCheckResult]{}
)

func lookupEpic(client *jira.Client, key string) *pending[epicCheckResult] {
	epicLookupsMu.Lock()
	defer epicLookupsMu.Unlock()
	if p, ok := epicLookups[key]; ok {
		return p
	}
	p := fetch(func() (epicCheckResult, error) {
		state, issue, err := client.CheckEpic(key)
		return epicCheckResult{state, issue, err}, nil
	})
	epicLookups[key] = p
	return p
}

// rememberEpic puts a known epic in the cache, e.g. right after it was picked.
func rememberEpic(epic *jiralib.Issue) {
	epicLookupsMu.Lock()
	defer epicLookupsMu.Unlock()
	epicLookups[epic.Key] = ready(epicCheckResult{state: jira.EpicOK, issue: epic})
}

// headerLookupTimeout keeps the menu from waiting on a slow Jira.
const headerLookupTimeout = 1500 * time.Millisecond

// defaultEpicLine describes the configured default epic for the menu header.
func defaultEpicLine(cfg *config.Config, client *jira.Client) string {
	key := cfg.IssueDefaults.EpicLink
	if key == "" {
		return "Default epic: none"
	}
	line := "Default epic: " + key
	if client == nil {
		return line
	}
	res, _, ok := lookupEpic(client, key).waitFor(headerLookupTimeout)
	if !ok {
		return line
	}
	switch res.state {
	case jira.EpicNotFound:
		return line + " (not found)"
	case jira.EpicNotAnEpic:
		return line + " (not an epic)"
	case jira.EpicOK:
		if res.issue != nil && jira.IsResolved(*res.issue) {
			return line + " (done)"
		}
		if s := epicSummary(res.issue); s != "" {
			return line + " " + textutil.Truncate(s, 40)
		}
	}
	return line
}

const noDefaultEpicRow = "None: no default epic"

// changeDefaultEpic is the "Change default epic" menu entry. It saves only
// issue_defaults.epic_link and returns what changed; Esc keeps the default.
func changeDefaultEpic() (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	client, err := jira.NewClient(cfg)
	if err != nil {
		return "", err
	}

	current := cfg.IssueDefaults.EpicLink
	var fixed []string
	if current != "" {
		label := current + " (current)"
		if res, _, ok := lookupEpic(client, current).waitFor(headerLookupTimeout); ok && res.issue != nil {
			label = epicPickLabel(*res.issue) + " (current)"
		}
		fixed = append(fixed, label)
	}
	fixed = append(fixed, noDefaultEpicRow)

	row, epic, err := chooseEpic(client, cfg.Project, "Change default epic (Esc keeps "+orNone(current)+")", fixed)
	switch {
	case tui.IsEsc(err):
		return "Default epic unchanged", nil
	case err != nil:
		return "", err
	}

	newKey := current
	switch {
	case epic != nil:
		newKey = epic.Key
	case fixed[row] == noDefaultEpicRow:
		newKey = ""
	}
	if newKey == current {
		return "Default epic unchanged", nil
	}

	cfg.IssueDefaults.EpicLink = newKey
	if err := cfg.Save(); err != nil {
		return "", fmt.Errorf("failed to save default epic: %w", err)
	}
	if newKey == "" {
		return "Default epic cleared", nil
	}
	rememberEpic(epic)
	return fmt.Sprintf("Default epic set to %s %s", newKey, textutil.Truncate(epicSummary(epic), 40)), nil
}
