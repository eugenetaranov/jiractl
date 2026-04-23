package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <ticket-url-or-key>",
	Short: "Inspect a Jira issue and print its full JSON",
	Long: `Fetch a Jira issue with expand=names,schema and pretty-print the raw JSON.
Useful for discovering customfield_XXXXX IDs and their values when setting up
[issue_defaults.custom_fields] in ~/.jiractl.toml.

Accepts a full browse URL or an issue key:
  jiractl inspect https://example.atlassian.net/browse/PROJ-123
  jiractl inspect PROJ-123`,
	Args: cobra.ExactArgs(1),
	RunE: runInspect,
}

func init() {
	RootCmd.AddCommand(inspectCmd)
}

var issueKeyRE = regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

func parseIssueKey(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty input")
	}
	match := issueKeyRE.FindString(strings.ToUpper(input))
	if match == "" {
		return "", fmt.Errorf("could not extract issue key from %q", input)
	}
	return match, nil
}

func runInspect(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	key, err := parseIssueKey(args[0])
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client, err := jira.NewClient(cfg)
	if err != nil {
		return err
	}

	if err := client.TestConnection(); err != nil {
		return fmt.Errorf("authentication failed — check credentials with 'jiractl configure': %w", err)
	}

	_, body, err := client.GetIssueRaw(key, debug)
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	return nil
}
