package cmd

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	jiralib "github.com/andygrunwald/go-jira"
	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/textutil"
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
// issue_defaults.epic_link; Esc leaves the config unchanged.
func changeDefaultEpic() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	client, err := jira.NewClient(cfg)
	if err != nil {
		return err
	}

	epics, err := client.GetEpics(cfg.Project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch epics: %v\n", err)
	}

	current := cfg.IssueDefaults.EpicLink
	type row struct {
		label string
		epic  *jiralib.Issue
		none  bool
	}
	var rows []row
	// The current default goes first so Enter keeps it.
	if current != "" {
		label := current + " (current)"
		var cur *jiralib.Issue
		for i := range epics {
			if epics[i].Key == current {
				cur = &epics[i]
				label = epicPickLabel(epics[i]) + " (current)"
			}
		}
		if cur == nil {
			if res, _, ok := lookupEpic(client, current).waitFor(headerLookupTimeout); ok && res.issue != nil {
				cur = res.issue
				label = epicPickLabel(*cur) + " (current)"
			}
		}
		rows = append(rows, row{label: label, epic: cur})
	}
	rows = append(rows, row{label: noDefaultEpicRow, none: true}, row{label: searchAllRow})
	for i := range epics {
		if epics[i].Key != current {
			rows = append(rows, row{label: epicPickLabel(epics[i]), epic: &epics[i]})
		}
	}

	labels := make([]string, len(rows))
	for i, r := range rows {
		labels[i] = r.label
	}
	idx, err := fzfSelect(labels, "Change default epic (Esc keeps "+orNone(current)+")")
	if errors.Is(err, ErrCancelled) {
		return nil
	}
	if err != nil {
		return err
	}

	chosen := rows[idx]
	var newKey string
	switch {
	case chosen.none:
		newKey = ""
	case chosen.label == searchAllRow:
		epic, err := searchEpic(client, cfg.Project)
		if err != nil || epic == nil {
			return err // Skip in the search keeps the current default
		}
		chosen.epic = epic
		newKey = epic.Key
	case chosen.epic != nil:
		newKey = chosen.epic.Key
	default:
		newKey = current
	}

	if newKey == current {
		menuStatus = "Default epic unchanged"
		return nil
	}
	cfg.IssueDefaults.EpicLink = newKey
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save default epic: %w", err)
	}

	if newKey == "" {
		menuStatus = "Default epic cleared"
	} else {
		rememberEpic(chosen.epic)
		menuStatus = fmt.Sprintf("Default epic set to %s %s", newKey, textutil.Truncate(epicSummary(chosen.epic), 40))
	}
	fmt.Fprintln(os.Stderr, menuStatus)
	return nil
}
