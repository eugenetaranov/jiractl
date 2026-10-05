package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	jiralib "github.com/andygrunwald/go-jira"
	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/keyring"
	"github.com/eugenetaranov/jiractl/internal/textutil"
	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure jiractl settings",
	Long: `Interactive setup for jiractl. Prompts for server URL, project key, username, and API token.
Nothing is saved until the connection, credentials and project have been verified.`,
	RunE: runConfigure,
}

func init() {
	RootCmd.AddCommand(configureCmd)
}

const maxConfigureAttempts = 3

// configureField names the value that failed validation and must be re-asked.
type configureField string

const (
	fieldServer  configureField = "server"
	fieldToken   configureField = "credentials"
	fieldProject configureField = "project"
)

type validationError struct {
	field configureField
	err   error
}

func (v *validationError) Error() string { return v.err.Error() }

func runConfigure(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	// Work on a copy; cfg stays as the last saved state until validation passes.
	updated := *cfg

	currentUsername, _ := keyring.GetUsername()
	existingToken, _ := keyring.GetToken()

	server, err := promptServer(cfg.Server)
	if err != nil {
		return err
	}
	project, err := promptTextWithDefault("Default Project Key", cfg.Project, true)
	if err != nil {
		return err
	}
	username, err := promptTextWithDefault("Username (email)", currentUsername, true)
	if err != nil {
		return err
	}
	tokenLabel := "API Token: "
	if existingToken != "" {
		tokenLabel = "API Token (leave empty to keep existing): "
	}
	token, err := readSecret(tokenLabel)
	if err != nil {
		return err
	}
	if token == "" && existingToken == "" {
		token, err = promptRequiredSecret("API Token: ")
		if err != nil {
			return err
		}
	}

	attempts := map[configureField]int{}
	var client *jira.Client
	var issueTypes []jiralib.IssueType
	for {
		updated.Server = server
		updated.Project = strings.ToUpper(project)
		effectiveToken := token
		if effectiveToken == "" {
			effectiveToken = existingToken
		}

		fmt.Fprint(os.Stderr, "\nTesting connection... ")
		client, issueTypes, err = validateSetup(&updated, username, effectiveToken)
		if err == nil {
			fmt.Fprintln(os.Stderr, "success!")
			break
		}
		fmt.Fprintln(os.Stderr, "failed")

		var verr *validationError
		if !errors.As(err, &verr) {
			return err
		}
		fmt.Fprintf(os.Stderr, "  %v\n", verr.err)
		attempts[verr.field]++
		if attempts[verr.field] >= maxConfigureAttempts {
			return fmt.Errorf("%s check failed %d times, nothing was saved: %w", verr.field, maxConfigureAttempts, verr.err)
		}

		switch verr.field {
		case fieldServer:
			server, err = promptServer(server)
		case fieldToken:
			username, err = promptTextWithDefault("Username (email)", username, true)
			if err == nil {
				token, err = promptRequiredSecret("API Token: ")
			}
		case fieldProject:
			project, err = promptTextWithDefault("Default Project Key", "", true)
		}
		if err != nil {
			return err
		}
	}

	// Optional defaults. Esc keeps the previous value.
	if len(issueTypes) > 0 {
		typeNames := make([]string, len(issueTypes))
		for i, it := range issueTypes {
			typeNames[i] = it.Name
		}
		idx, err := fzfSelect(typeNames, "Select default issue type (Esc keeps current)")
		switch {
		case err == nil:
			updated.IssueDefaults.IssueType = typeNames[idx]
		case !errors.Is(err, ErrCancelled):
			return err
		}
	}

	epics, err := client.GetEpics(updated.Project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch epics: %v\n", err)
	} else if len(epics) > 0 {
		epicItems := make([]string, len(epics)+1)
		epicItems[0] = "(None)"
		for i, epic := range epics {
			epicItems[i+1] = epicLabel(epic)
		}
		idx, err := fzfSelect(epicItems, "Select default epic (Esc keeps current)")
		switch {
		case err == nil && idx > 0:
			updated.IssueDefaults.EpicLink = epics[idx-1].Key
		case err == nil:
			updated.IssueDefaults.EpicLink = ""
		case !errors.Is(err, ErrCancelled):
			return err
		}
	}

	// Everything checked out: persist once.
	if err := keyring.SetUsername(username); err != nil {
		return fmt.Errorf("failed to save username: %w", err)
	}
	if token != "" {
		if err := keyring.SetToken(token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}
	}
	if err := updated.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("\nConfiguration saved to ~/.jiractl.toml")
	fmt.Printf("  Server:             %s\n", updated.Server)
	fmt.Printf("  Project:            %s\n", updated.Project)
	fmt.Printf("  Default issue type: %s\n", orNone(updated.IssueDefaults.IssueType))
	fmt.Printf("  Default epic:       %s\n", orNone(updated.IssueDefaults.EpicLink))
	if token != "" {
		fmt.Println("  Credentials:        stored in system keyring")
	}
	return nil
}

// validateSetup checks server, credentials and project using in-memory values.
func validateSetup(cfg *config.Config, username, token string) (*jira.Client, []jiralib.IssueType, error) {
	client, err := jira.NewClientWith(cfg, username, token)
	if err != nil {
		return nil, nil, &validationError{fieldServer, err}
	}

	if err := client.TestConnection(); err != nil {
		switch jira.StatusOf(err) {
		case 401, 403:
			return nil, nil, &validationError{fieldToken, fmt.Errorf("credentials rejected: %w", err)}
		default:
			return nil, nil, &validationError{fieldServer, fmt.Errorf("cannot reach Jira at %s: %w", cfg.Server, err)}
		}
	}

	issueTypes, err := client.GetIssueTypes(cfg.Project)
	if err != nil {
		if jira.StatusOf(err) == 404 {
			return nil, nil, &validationError{fieldProject, fmt.Errorf("project %s not found or not visible to you", cfg.Project)}
		}
		return nil, nil, &validationError{fieldProject, fmt.Errorf("cannot load project %s: %w", cfg.Project, err)}
	}
	return client, issueTypes, nil
}

func promptServer(current string) (string, error) {
	for {
		server, err := promptTextWithDefault("Jira Server URL", current, true)
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(server, "http://") || strings.HasPrefix(server, "https://") {
			return strings.TrimRight(server, "/"), nil
		}
		fmt.Fprintln(os.Stderr, "Server URL must start with http:// or https://")
	}
}

func promptRequiredSecret(label string) (string, error) {
	for {
		token, err := readSecret(label)
		if err != nil || token != "" {
			return token, err
		}
		fmt.Fprintln(os.Stderr, "This field is required")
	}
}

func epicLabel(epic jiralib.Issue) string {
	summary := ""
	if epic.Fields != nil {
		summary = epic.Fields.Summary
	}
	return fmt.Sprintf("%s - %s", epic.Key, textutil.Truncate(summary, 50))
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}
