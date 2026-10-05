package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	jiralib "github.com/andygrunwald/go-jira"
	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/textutil"
	"github.com/eugenetaranov/jiractl/internal/tui"
)

const skipEpicRow = "Skip: create without epic"

func epicPickLabel(epic jiralib.Issue) string {
	label := epicLabel(epic)
	if jira.IsResolved(epic) {
		label += " [done]"
	}
	return label
}

func epicSummary(epic *jiralib.Issue) string {
	if epic == nil || epic.Fields == nil {
		return ""
	}
	return epic.Fields.Summary
}

// chooseEpic opens the epic search, which asks Jira as the user types.
// fixed rows are listed first; it returns the chosen fixed row (or -1) and
// the chosen epic (or nil).
func chooseEpic(client *jira.Client, project, header string, fixed []string) (int, *jiralib.Issue, error) {
	var mu sync.Mutex
	byQuery := map[string][]jiralib.Issue{}
	res, err := tui.SearchSelect(tui.SearchOptions{
		Header: header,
		Fixed:  fixed,
		Search: func(q string) ([]string, error) {
			epics, err := client.SearchEpics(project, q)
			if err != nil {
				return nil, err
			}
			mu.Lock()
			byQuery[q] = epics
			mu.Unlock()
			labels := make([]string, len(epics))
			for i, e := range epics {
				labels[i] = epicPickLabel(e)
			}
			return labels, nil
		},
	})
	if err != nil {
		return -1, nil, err
	}
	if res.Fixed >= 0 {
		return res.Fixed, nil, nil
	}
	mu.Lock()
	defer mu.Unlock()
	epic := byQuery[res.Query][res.Index]
	return -1, &epic, nil
}

// searchEpic lets the user find another epic or skip it (nil). Esc skips.
func searchEpic(client *jira.Client, project string) (*jiralib.Issue, error) {
	_, epic, err := chooseEpic(client, project, "Search epics (type words or a key)", []string{skipEpicRow})
	if tui.IsEsc(err) {
		return nil, nil
	}
	return epic, err
}

// pickEpic is the regular epic choice: open epics, narrowed by searching
// Jira as the user types. "(None)" returns nil; Esc is returned to the
// caller, which decides whether it means "no epic" or "keep it".
func pickEpic(client *jira.Client, project string) (*jiralib.Issue, error) {
	_, epic, err := chooseEpic(client, project, "Select epic (optional)", []string{"(None)"})
	return epic, err
}

// recoverEpic runs when the chosen epic can't be used. Interactively it lets
// the user search for another epic or skip; otherwise it returns an error
// listing candidates.
func recoverEpic(cfg *config.Config, client *jira.Client, draft *issueDraft, problem string, fromDefault, interactive bool) (*jiralib.Issue, error) {
	bad := draft.EpicLink
	if !interactive {
		return nil, epicCandidatesError(client, cfg.Project, bad, problem)
	}

	fmt.Fprintf(os.Stderr, "\nEpic %s %s. Pick another epic or skip it.\n", bad, problem)
	epic, err := searchEpic(client, cfg.Project)
	if err != nil {
		return nil, err
	}
	if epic == nil {
		draft.EpicLink = ""
		return nil, nil
	}
	draft.EpicLink = epic.Key

	if fromDefault {
		save, err := promptConfirm(fmt.Sprintf("Save %s as default epic?", epic.Key), false)
		if err != nil {
			return nil, err
		}
		if save {
			cfg.IssueDefaults.EpicLink = epic.Key
			if err := cfg.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not save default epic: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Default epic set to %s\n", epic.Key)
			}
		}
	}
	return epic, nil
}

func epicCandidatesError(client *jira.Client, project, key, problem string) error {
	msg := fmt.Sprintf("epic %s %s", key, problem)
	if epics, err := client.SearchEpics(project, ""); err == nil && len(epics) > 0 {
		var names []string
		for i, e := range epics {
			if i == 5 {
				break
			}
			names = append(names, fmt.Sprintf("%s %s", e.Key, textutil.Truncate(epicSummary(&epics[i]), 40)))
		}
		msg += "; candidates: " + strings.Join(names, ", ")
	}
	return errors.New(msg + " (use --no-epic to create without an epic)")
}

// epicProblem describes why an epic can't be used, or "" when it can.
func epicProblem(state jira.EpicState) string {
	switch state {
	case jira.EpicNotFound:
		return "not found"
	case jira.EpicNotAnEpic:
		return "is not an epic"
	}
	return ""
}
