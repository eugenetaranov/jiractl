package cmd

import (
	"fmt"
	"strings"

	"github.com/e/jiractl/internal/config"
	"github.com/e/jiractl/internal/jira"
	"github.com/manifoldco/promptui"
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

	// Find default issue type index
	defaultIdx := 0
	if cfg.IssueDefaults.IssueType != "" {
		for i, name := range typeNames {
			if name == cfg.IssueDefaults.IssueType {
				defaultIdx = i
				break
			}
		}
	}

	// Prompt for issue type
	typePrompt := promptui.Select{
		Label:     "Issue Type",
		Items:     typeNames,
		CursorPos: defaultIdx,
	}
	_, issueType, err := typePrompt.Run()
	if err != nil {
		return handlePromptError(err)
	}

	// Prompt for summary
	summaryPrompt := promptui.Prompt{
		Label: "Summary",
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("summary is required")
			}
			return nil
		},
	}
	summary, err := summaryPrompt.Run()
	if err != nil {
		return handlePromptError(err)
	}

	// Prompt for description
	descPrompt := promptui.Prompt{
		Label: "Description (optional)",
	}
	description, err := descPrompt.Run()
	if err != nil {
		return handlePromptError(err)
	}

	// Confirm creation
	fmt.Printf("\nCreating issue:\n")
	fmt.Printf("  Project:     %s\n", cfg.Project)
	fmt.Printf("  Type:        %s\n", issueType)
	fmt.Printf("  Summary:     %s\n", summary)
	if description != "" {
		fmt.Printf("  Description: %s\n", description)
	}

	confirmPrompt := promptui.Prompt{
		Label:     "Create this issue",
		IsConfirm: true,
	}
	_, err = confirmPrompt.Run()
	if err != nil {
		fmt.Println("Issue creation cancelled.")
		return nil
	}

	// Create the issue
	issue, err := client.CreateIssue(cfg.Project, issueType, summary, description)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	fmt.Printf("\nCreated issue: %s\n", issue.Key)
	fmt.Printf("URL: %s/browse/%s\n", cfg.Server, issue.Key)

	return nil
}
