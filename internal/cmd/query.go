package cmd

import (
	"fmt"
	"os"

	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/textutil"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query [name]",
	Short: "Run a saved query",
	Long:  `Run a saved JQL query from your config file and display results interactively.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runQueryCmd,
}

func init() {
	RootCmd.AddCommand(queryCmd)
}

func runQueryCmd(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return runQueryInteractive()
	}
	return runQuery(args[0])
}

func runQuery(queryName string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	query := cfg.GetQuery(queryName)
	if query == nil {
		return fmt.Errorf("query not found: %s", queryName)
	}

	client, err := jira.NewClient(cfg)
	if err != nil {
		return err
	}

	// Expand variables in JQL
	jql := cfg.ExpandJQL(query.JQL)

	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}

	fmt.Fprintf(os.Stderr, "Running query: %s\n", queryName)
	fmt.Fprintf(os.Stderr, "JQL: %s\n\n", jql)

	issues, err := client.SearchIssues(jql, limit)
	if err != nil {
		return fmt.Errorf("query %q failed: %w", queryName, err)
	}

	if len(issues) == 0 {
		fmt.Fprintln(os.Stderr, "No issues found.")
		return nil
	}

	// Display issues in a select menu
	items := make([]string, len(issues))
	for i, issue := range issues {
		status := ""
		if issue.Fields != nil && issue.Fields.Status != nil {
			status = issue.Fields.Status.Name
		}
		summary := ""
		if issue.Fields != nil {
			summary = issue.Fields.Summary
		}
		items[i] = textutil.PadRight(issue.Key, 12) + " " + textutil.PadRight(status, 15) + " " + textutil.Truncate(summary, 60)
	}

	idx, err := fzfSelect(items, fmt.Sprintf("Select issue (%d found)", len(issues)))
	if err != nil {
		return err
	}

	// Show selected issue details
	selected := issues[idx]
	return showIssueDetails(client, cfg.Server, selected.Key)
}

func showIssueDetails(client *jira.Client, server, key string) error {
	issue, err := client.GetIssue(key)
	if err != nil {
		return fmt.Errorf("failed to get issue %s: %w", key, err)
	}

	fmt.Printf("\n%s: %s\n", issue.Key, issue.Fields.Summary)
	fmt.Printf("%s/browse/%s\n", server, issue.Key)
	fmt.Printf("─────────────────────────────────────────────────────────\n")

	if issue.Fields.Status != nil {
		fmt.Printf("Status:      %s\n", issue.Fields.Status.Name)
	}
	if issue.Fields.Type.Name != "" {
		fmt.Printf("Type:        %s\n", issue.Fields.Type.Name)
	}
	if issue.Fields.Priority != nil {
		fmt.Printf("Priority:    %s\n", issue.Fields.Priority.Name)
	}
	if issue.Fields.Assignee != nil {
		fmt.Printf("Assignee:    %s\n", issue.Fields.Assignee.DisplayName)
	}
	if issue.Fields.Reporter != nil {
		fmt.Printf("Reporter:    %s\n", issue.Fields.Reporter.DisplayName)
	}
	if len(issue.Fields.Labels) > 0 {
		fmt.Printf("Labels:      %v\n", issue.Fields.Labels)
	}
	if issue.Fields.Description != "" {
		fmt.Printf("\nDescription:\n%s\n", issue.Fields.Description)
	}

	return nil
}
