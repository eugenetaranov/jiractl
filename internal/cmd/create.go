package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/textutil"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Jira issue",
	Long:  `Interactively create a new Jira issue with summary, description, and other fields.`,
	RunE:  runCreate,
}

func init() {
	RootCmd.AddCommand(createCmd)
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if cfg.Server == "" || cfg.Project == "" {
		return nil, fmt.Errorf("not configured, run 'jiractl configure' first")
	}
	return cfg, nil
}

func runCreate(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client, err := jira.NewClient(cfg)
	if err != nil {
		return err
	}

	draft := &issueDraft{Project: cfg.Project}
	resumed := false
	if saved := loadDraft(); saved != nil && saved.Project == cfg.Project {
		label := fmt.Sprintf("Resume draft %q from %s?", textutil.Truncate(saved.Summary, 40), saved.SavedAt.Format("Jan 2 15:04"))
		resume, err := promptConfirm(label, true)
		if err != nil {
			return err
		}
		if resume {
			draft, resumed = saved, true
		} else {
			deleteDraft()
		}
	}

	if !resumed {
		if err := promptNewIssue(cfg, client, draft); err != nil {
			return err
		}
	}

	epicSummary := ""
	if draft.EpicLink != "" {
		if epic, err := client.GetIssue(draft.EpicLink); err == nil && epic.Fields != nil {
			epicSummary = epic.Fields.Summary
		}
	}

	printReview(cfg, draft, epicSummary)

	confirmed, err := promptConfirm("Create this issue?", true)
	if err != nil {
		keepDraft(draft)
		return err
	}
	if !confirmed {
		keepDraft(draft)
		return ErrCancelled
	}

	opts := &jira.CreateIssueOptions{EpicLink: draft.EpicLink}
	issue, err := client.CreateIssue(cfg.Project, draft.IssueType, draft.Summary, draft.Description, opts)
	if err != nil {
		keepDraft(draft)
		return fmt.Errorf("failed to create issue: %w", err)
	}
	deleteDraft()

	fmt.Printf("\nCreated issue: %s\n", issue.Key)
	fmt.Printf("%s/browse/%s\n", cfg.Server, issue.Key)

	return nil
}

// promptNewIssue asks for type, summary, description and epic.
func promptNewIssue(cfg *config.Config, client *jira.Client, draft *issueDraft) error {
	if cfg.IssueDefaults.IssueType != "" {
		draft.IssueType = cfg.IssueDefaults.IssueType
	} else {
		issueTypes, err := client.GetIssueTypes(cfg.Project)
		if err != nil {
			return fmt.Errorf("failed to get issue types: %w", err)
		}
		typeNames := make([]string, len(issueTypes))
		for i, it := range issueTypes {
			typeNames[i] = it.Name
		}
		idx, err := fzfSelect(typeNames, "Select issue type")
		if err != nil {
			return err
		}
		draft.IssueType = typeNames[idx]
	}

	summary, err := promptText("Summary", true)
	if err != nil {
		return err
	}
	draft.Summary = summary

	description, err := promptMultilineText("Description (optional)")
	if err != nil {
		return err
	}
	draft.Description = description

	if cfg.IssueDefaults.EpicLink != "" {
		draft.EpicLink = cfg.IssueDefaults.EpicLink
		return nil
	}

	epics, err := client.GetEpics(cfg.Project)
	if err != nil {
		// Non-fatal: just skip epic selection
		fmt.Fprintf(os.Stderr, "Warning: could not fetch epics: %v\n", err)
		return nil
	}
	if len(epics) == 0 {
		return nil
	}
	epicItems := make([]string, len(epics)+1)
	epicItems[0] = "(None)"
	for i, epic := range epics {
		epicItems[i+1] = epicLabel(epic)
	}
	idx, err := fzfSelect(epicItems, "Select epic (optional)")
	if err != nil {
		return err
	}
	if idx > 0 {
		draft.EpicLink = epics[idx-1].Key
	}
	return nil
}

func printReview(cfg *config.Config, draft *issueDraft, epicSummary string) {
	w := os.Stderr
	fmt.Fprintf(w, "\nCreating issue:\n")
	fmt.Fprintf(w, "  Project:     %s\n", cfg.Project)
	fmt.Fprintf(w, "  Type:        %s\n", draft.IssueType)
	fmt.Fprintf(w, "  Summary:     %s\n", draft.Summary)
	if draft.Description != "" {
		lines := strings.Split(draft.Description, "\n")
		if len(lines) == 1 && textutil.Width(draft.Description) <= 50 {
			fmt.Fprintf(w, "  Description: %s\n", draft.Description)
		} else {
			fmt.Fprintf(w, "  Description: (%d lines)\n", len(lines))
		}
	}
	if draft.EpicLink != "" {
		if epicSummary != "" {
			fmt.Fprintf(w, "  Epic:        %s - %s\n", draft.EpicLink, epicSummary)
		} else {
			fmt.Fprintf(w, "  Epic:        %s\n", draft.EpicLink)
		}
	}
	d := cfg.IssueDefaults
	if d.Assignee != "" {
		fmt.Fprintf(w, "  Assignee:    %s\n", d.Assignee)
	}
	if d.Component != "" {
		fmt.Fprintf(w, "  Component:   %s\n", d.Component)
	}
	if len(d.Labels) > 0 {
		fmt.Fprintf(w, "  Labels:      %s\n", strings.Join(d.Labels, ", "))
	}
	for k, v := range d.CustomFields {
		fmt.Fprintf(w, "  %s: %s\n", k, v)
	}
}
