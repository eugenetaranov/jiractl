package cmd

import (
	"fmt"

	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
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
	cmd.SilenceUsage = true

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client, err := jira.NewClient(cfg)
	if err != nil {
		return err
	}

	// Determine issue type
	var issueType string
	if cfg.IssueDefaults.IssueType != "" {
		issueType = cfg.IssueDefaults.IssueType
	} else {
		// Get available issue types
		issueTypes, err := client.GetIssueTypes(cfg.Project)
		if err != nil {
			return fmt.Errorf("failed to get issue types: %w", err)
		}

		// Build issue type list
		typeNames := make([]string, len(issueTypes))
		for i, it := range issueTypes {
			typeNames[i] = it.Name
		}

		// Prompt for issue type
		idx, err := fzfSelect(typeNames, "Select issue type")
		if err != nil {
			if err == fuzzyfinder.ErrAbort {
				fmt.Println("\nCancelled.")
				return nil
			}
			return err
		}
		issueType = typeNames[idx]
	}

	// Prompt for summary
	summary, err := promptText("Summary", true)
	if err != nil {
		if err == ErrPromptCancelled {
			fmt.Println("\nCancelled.")
			return nil
		}
		return err
	}

	// Prompt for description
	description, err := promptText("Description (optional)", false)
	if err != nil {
		if err == ErrPromptCancelled {
			fmt.Println("\nCancelled.")
			return nil
		}
		return err
	}

	// Determine epic link
	var epicLink string
	if cfg.IssueDefaults.EpicLink != "" {
		epicLink = cfg.IssueDefaults.EpicLink
	} else {
		// Prompt for epic if not configured
		epics, err := client.GetEpics(cfg.Project)
		if err != nil {
			// Non-fatal: just skip epic selection
			fmt.Printf("Warning: could not fetch epics: %v\n", err)
		} else if len(epics) > 0 {
			epicItems := make([]string, len(epics)+1)
			epicItems[0] = "(None)"
			for i, epic := range epics {
				epicSummary := ""
				if epic.Fields != nil {
					epicSummary = epic.Fields.Summary
				}
				if len(epicSummary) > 50 {
					epicSummary = epicSummary[:47] + "..."
				}
				epicItems[i+1] = fmt.Sprintf("%s - %s", epic.Key, epicSummary)
			}

			idx, err := fzfSelect(epicItems, "Select epic (optional)")
			if err != nil {
				if err == fuzzyfinder.ErrAbort {
					fmt.Println("\nCancelled.")
					return nil
				}
				return err
			}
			if idx > 0 {
				epicLink = epics[idx-1].Key
			}
		}
	}

	// Confirm creation
	fmt.Printf("\nCreating issue:\n")
	fmt.Printf("  Project:     %s\n", cfg.Project)
	fmt.Printf("  Type:        %s\n", issueType)
	fmt.Printf("  Summary:     %s\n", summary)
	if description != "" {
		fmt.Printf("  Description: %s\n", description)
	}
	if epicLink != "" {
		fmt.Printf("  Epic:        %s\n", epicLink)
	}

	confirmed, err := promptConfirm("Create this issue?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Issue creation cancelled.")
		return nil
	}

	// Create the issue
	opts := &jira.CreateIssueOptions{EpicLink: epicLink}
	issue, err := client.CreateIssue(cfg.Project, issueType, summary, description, opts)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	fmt.Printf("\nCreated issue: %s\n", issue.Key)
	fmt.Printf("%s/browse/%s\n", cfg.Server, issue.Key)

	return nil
}
