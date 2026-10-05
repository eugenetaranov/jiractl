package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	jiralib "github.com/andygrunwald/go-jira"
	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/textutil"
	"golang.org/x/term"
)

const (
	skipEpicRow    = "Skip: create without epic"
	searchAgainRow = "Search again…"
	searchAllRow   = "Search all epics…"
)

// isInteractive reports whether prompts can be shown.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stderr.Fd()))
}

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

// searchEpic asks for search words, then shows the matching epics from Jira.
// It returns the chosen epic, or nil when the user skips.
func searchEpic(client *jira.Client, project string) (*jiralib.Issue, error) {
	text := ""
	for {
		q, err := promptTextWithDefault("Search epics (words or key, Enter for open epics)", text, false)
		if err != nil {
			return nil, err
		}
		text = q

		epics, err := client.SearchEpics(project, q)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Epic search failed: %v\n", err)
		}
		items := []string{skipEpicRow, searchAgainRow}
		for _, e := range epics {
			items = append(items, epicPickLabel(e))
		}
		header := fmt.Sprintf("%d epics match %q", len(epics), q)
		if q == "" {
			header = fmt.Sprintf("%d open epics", len(epics))
		}

		idx, err := fzfSelect(items, header)
		switch {
		case errors.Is(err, ErrCancelled), err == nil && idx == 0:
			return nil, nil
		case err != nil:
			return nil, err
		case idx == 1:
			continue
		}
		return &epics[idx-2], nil
	}
}

// pickEpic is the regular epic choice when no default is configured: recent
// open epics, plus a row that searches all epics in Jira.
func pickEpic(client *jira.Client, project string) (*jiralib.Issue, error) {
	epics, err := client.GetEpics(project)
	if err != nil {
		// Non-fatal: just skip epic selection
		fmt.Fprintf(os.Stderr, "Warning: could not fetch epics: %v\n", err)
		return nil, nil
	}
	items := []string{"(None)", searchAllRow}
	for _, e := range epics {
		items = append(items, epicPickLabel(e))
	}
	idx, err := fzfSelect(items, "Select epic (optional)")
	if err != nil {
		return nil, err
	}
	switch idx {
	case 0:
		return nil, nil
	case 1:
		return searchEpic(client, project)
	}
	return &epics[idx-2], nil
}

// recoverEpic runs when the chosen epic can't be used. Interactively it lets
// the user search for another epic or skip; otherwise it returns an error
// listing candidates.
func recoverEpic(cfg *config.Config, client *jira.Client, draft *issueDraft, problem string, fromDefault bool) (*jiralib.Issue, error) {
	bad := draft.EpicLink
	if !isInteractive() {
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
