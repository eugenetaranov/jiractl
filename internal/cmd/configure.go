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
	Long: `Interactive setup for jiractl: server URL, credentials, project, and default
issue type and epic. Each value is checked as soon as it is entered, and
nothing is saved until every step has passed.`,
	RunE: runConfigure,
}

func init() {
	RootCmd.AddCommand(configureCmd)
}

const maxConfigureAttempts = 3

func runConfigure(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	// Work on a copy; nothing is written until every step passed.
	updated := *cfg

	server, err := askServer(cfg.Server)
	if err != nil {
		return err
	}
	updated.Server = server

	creds, err := promptCredentials(&updated)
	if err != nil {
		return err
	}

	project, issueTypes, err := askProject(creds.client, &updated, cfg.Project)
	if err != nil {
		return err
	}
	updated.Project = project

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

	epics, err := creds.client.GetEpics(updated.Project)
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

	addedQueries := false
	if len(updated.Queries) == 0 {
		updated.Queries = append([]config.Query(nil), config.StarterQueries...)
		addedQueries = true
	}

	// Everything checked out: persist once.
	if err := creds.save(); err != nil {
		return err
	}
	if err := updated.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("\nConfiguration saved to ~/.jiractl.toml")
	fmt.Printf("  Server:             %s\n", updated.Server)
	fmt.Printf("  Project:            %s\n", updated.Project)
	fmt.Printf("  Default issue type: %s\n", orNone(updated.IssueDefaults.IssueType))
	fmt.Printf("  Default epic:       %s\n", orNone(updated.IssueDefaults.EpicLink))
	if creds.token != "" {
		fmt.Println("  Credentials:        stored in system keyring")
	}
	if addedQueries {
		fmt.Printf("  Queries:            added %s (try 'jiractl query mine')\n", strings.Join(updated.QueryNames(), ", "))
	}
	menuStatus = "Configuration saved"
	return nil
}

// normalizeServer adds https:// when no scheme is given and drops trailing
// slashes.
func normalizeServer(s string) string {
	s = strings.TrimRight(strings.TrimSpace(s), "/")
	if s != "" && !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		s = "https://" + s
	}
	return s
}

// askServer asks for the server URL and checks it is a reachable Jira.
func askServer(current string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < maxConfigureAttempts; attempt++ {
		input, err := promptTextWithDefault("Jira Server URL", current, true)
		if err != nil {
			return "", err
		}
		server := normalizeServer(input)

		client, err := jira.NewClientWith(&config.Config{Server: server}, "", "")
		if err == nil {
			var info *jira.ServerInfo
			if info, err = client.GetServerInfo(); err == nil {
				fmt.Fprintf(os.Stderr, "  Found %s %s\n", orDefault(info.DeploymentType, "Jira"), info.Version)
				return server, nil
			}
		}
		lastErr = fmt.Errorf("cannot reach Jira at %s: %w", server, err)
		fmt.Fprintf(os.Stderr, "  %v\n", lastErr)
		current = server
	}
	return "", fmt.Errorf("server check failed %d times, nothing was saved: %w", maxConfigureAttempts, lastErr)
}

// credentials are checked username/token values, saved only on request.
type credentials struct {
	username string
	token    string // empty when the stored token is kept
	client   *jira.Client
}

func (c *credentials) save() error {
	if err := keyring.SetUsername(c.username); err != nil {
		return fmt.Errorf("failed to save username: %w", err)
	}
	if c.token != "" {
		if err := keyring.SetToken(c.token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}
	}
	return nil
}

// promptCredentials asks for username and API token and checks them against
// the server right away. It is shared by configure and auth create.
func promptCredentials(cfg *config.Config) (*credentials, error) {
	currentUsername, _ := keyring.GetUsername()
	existingToken, _ := keyring.GetToken()

	var lastErr error
	for attempt := 0; attempt < maxConfigureAttempts; attempt++ {
		username, err := promptTextWithDefault("Username (email)", currentUsername, true)
		if err != nil {
			return nil, err
		}

		label := "API Token: "
		if existingToken != "" && attempt == 0 {
			label = "API Token (leave empty to keep existing): "
		}
		token, err := readSecret(label)
		if err != nil {
			return nil, err
		}
		effective := token
		if effective == "" {
			effective = existingToken
		}
		if effective == "" {
			fmt.Fprintln(os.Stderr, "  API token is required (create one at https://id.atlassian.com/manage-profile/security/api-tokens)")
			continue
		}

		client, err := jira.NewClientWith(cfg, username, effective)
		if err != nil {
			return nil, err
		}
		fmt.Fprint(os.Stderr, "  Checking credentials... ")
		err = client.TestConnection()
		if err == nil {
			fmt.Fprintln(os.Stderr, "ok")
			return &credentials{username: username, token: token, client: client}, nil
		}
		fmt.Fprintln(os.Stderr, "failed")
		switch jira.StatusOf(err) {
		case 401, 403:
			lastErr = fmt.Errorf("credentials rejected: %w", err)
			fmt.Fprintf(os.Stderr, "  %v\n", lastErr)
			currentUsername = username
			existingToken = "" // a rejected token can't be kept
		default:
			return nil, fmt.Errorf("cannot reach Jira at %s: %w", cfg.Server, err)
		}
	}
	return nil, fmt.Errorf("credentials check failed %d times, nothing was saved: %w", maxConfigureAttempts, lastErr)
}

// askProject lets the user pick a project from the ones they can see, with
// the current one listed first. Typing filters by key or name.
func askProject(client *jira.Client, cfg *config.Config, current string) (string, []jiralib.IssueType, error) {
	projects, err := client.ListProjects()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not list projects: %v\n", err)
	}

	var lastErr error
	for attempt := 0; attempt < maxConfigureAttempts; attempt++ {
		var key string
		if len(projects) > 0 {
			items := make([]string, 0, len(projects))
			order := make([]jira.Project, 0, len(projects))
			for _, p := range projects {
				if strings.EqualFold(p.Key, current) {
					order = append([]jira.Project{p}, order...)
				} else {
					order = append(order, p)
				}
			}
			for _, p := range order {
				items = append(items, fmt.Sprintf("%s - %s", p.Key, textutil.Truncate(p.Name, 60)))
			}
			header := "Select project"
			if current != "" {
				header += " (current: " + current + ")"
			}
			idx, err := fzfSelect(items, header)
			if errors.Is(err, ErrCancelled) && current != "" {
				key = current
			} else if err != nil {
				return "", nil, err
			} else {
				key = order[idx].Key
			}
		} else {
			key, err = promptTextWithDefault("Default Project Key", current, true)
			if err != nil {
				return "", nil, err
			}
		}
		key = strings.ToUpper(strings.TrimSpace(key))

		issueTypes, err := client.GetIssueTypes(key)
		if err == nil {
			return key, issueTypes, nil
		}
		if jira.StatusOf(err) == 404 {
			lastErr = fmt.Errorf("project %s not found or not visible to you", key)
		} else {
			lastErr = fmt.Errorf("cannot load project %s: %w", key, err)
		}
		fmt.Fprintf(os.Stderr, "  %v\n", lastErr)
		current = ""
	}
	return "", nil, fmt.Errorf("project check failed %d times, nothing was saved: %w", maxConfigureAttempts, lastErr)
}

func epicLabel(epic jiralib.Issue) string {
	summary := ""
	if epic.Fields != nil {
		summary = epic.Fields.Summary
	}
	return fmt.Sprintf("%s - %s", epic.Key, textutil.Truncate(summary, 50))
}

func orNone(s string) string {
	return orDefault(s, "(none)")
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
