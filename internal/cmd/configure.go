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
	"github.com/eugenetaranov/jiractl/internal/tui"
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

	server, deployment, err := askServer(cfg.Server)
	if err != nil {
		return err
	}
	updated.Server = server
	updated.Deployment = deployment

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
		case !tui.IsEsc(err):
			return err
		}
	}

	// Default epic, searched live; Esc keeps the current one.
	_, epic, err := chooseEpic(creds.client, updated.Project, "Select default epic (Esc keeps current)", []string{"(None)"})
	switch {
	case err == nil:
		updated.IssueDefaults.EpicLink = ""
		if epic != nil {
			updated.IssueDefaults.EpicLink = epic.Key
		}
	case !tui.IsEsc(err):
		return err
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
	runPostConfigureDoctor()
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

// askServer asks for the server URL and checks it is a reachable Jira while
// the field is still open. It returns the URL and the deployment type.
func askServer(current string) (string, string, error) {
	var server string
	var info *jira.ServerInfo
	_, err := tui.Input("Jira Server URL", tui.InputOptions{
		Default:     current,
		Required:    true,
		CheckLabel:  "Checking server",
		MaxAttempts: maxConfigureAttempts,
		Check: func(v string) error {
			s := normalizeServer(v)
			client, err := jira.NewClientWith(&config.Config{Server: s}, "", "")
			if err == nil {
				info, err = client.GetServerInfo()
			}
			if err != nil {
				return fmt.Errorf("cannot reach Jira at %s: %w", s, err)
			}
			server = s
			return nil
		},
	})
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return "", "", err
		}
		return "", "", fmt.Errorf("server check failed %d times, nothing was saved: %w", maxConfigureAttempts, err)
	}
	fmt.Fprintf(os.Stderr, "  Found %s %s\n", orDefault(info.DeploymentType, "Jira"), info.Version)
	deployment := config.DeploymentCloud
	if info.DeploymentType != "" && !strings.EqualFold(info.DeploymentType, "Cloud") {
		deployment = config.DeploymentServer
	}
	return server, deployment, nil
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
// the server before moving on. It is shared by configure and auth create.
// A rejected token re-asks both values (the username may be the problem).
func promptCredentials(cfg *config.Config) (*credentials, error) {
	currentUsername, _ := keyring.GetUsername()
	existingToken, _ := keyring.GetToken()

	tokenName, tokenHelp := "API Token", "create one at https://id.atlassian.com/manage-profile/security/api-tokens"
	if cfg.IsServer() {
		tokenName, tokenHelp = "Personal Access Token", "Profile → Personal Access Tokens in Jira"
	}

	var lastErr error
	for attempt := 0; attempt < maxConfigureAttempts; attempt++ {
		// Server/Data Center authenticates with a personal access token alone.
		username := ""
		if !cfg.IsServer() {
			var err error
			username, err = promptTextWithDefault("Username (email)", currentUsername, true)
			if err != nil {
				return nil, err
			}
		}

		opts := tui.InputOptions{Required: existingToken == "", Hint: "(" + tokenHelp + ")", CheckLabel: "Checking credentials", MaxAttempts: 1}
		if existingToken != "" {
			opts.Hint = "(leave empty to keep the stored one)"
		}
		var client *jira.Client
		opts.Check = func(token string) error {
			effective := token
			if effective == "" {
				effective = existingToken
			}
			c, err := jira.NewClientWith(cfg, username, effective)
			if err != nil {
				return err
			}
			if err := c.TestConnection(); err != nil {
				if s := jira.StatusOf(err); s == 401 || s == 403 {
					return fmt.Errorf("credentials rejected: %w", err)
				}
				return fmt.Errorf("cannot reach Jira at %s: %w", cfg.Server, err)
			}
			client = c
			return nil
		}
		token, err := tui.Secret(tokenName, opts)
		if err == nil {
			return &credentials{username: username, token: token, client: client}, nil
		}
		if errors.Is(err, ErrCancelled) {
			return nil, err
		}
		lastErr = err
		currentUsername = username
		existingToken = "" // a rejected token can't be kept
	}
	return nil, fmt.Errorf("credentials check failed %d times, nothing was saved: %w", maxConfigureAttempts, lastErr)
}

// askProject lets the user pick a project from the ones they can see, with
// the current one listed first, and checks it while the list is open. Esc
// keeps the current project.
func askProject(client *jira.Client, cfg *config.Config, current string) (string, []jiralib.IssueType, error) {
	var issueTypes []jiralib.IssueType
	check := func(key string) error {
		types, err := client.GetIssueTypes(key)
		if err != nil {
			if jira.StatusOf(err) == 404 {
				return fmt.Errorf("project %s not found or not visible to you", key)
			}
			return fmt.Errorf("cannot load project %s: %w", key, err)
		}
		issueTypes = types
		return nil
	}

	projects, err := client.ListProjects()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not list projects: %v\n", err)
	}

	if len(projects) == 0 {
		key, err := tui.Input("Default Project Key", tui.InputOptions{
			Default: current, Required: true, CheckLabel: "Checking project", MaxAttempts: maxConfigureAttempts,
			Check: func(v string) error { return check(strings.ToUpper(v)) },
		})
		if err != nil {
			if errors.Is(err, ErrCancelled) {
				return "", nil, err
			}
			return "", nil, fmt.Errorf("project check failed %d times, nothing was saved: %w", maxConfigureAttempts, err)
		}
		return strings.ToUpper(key), issueTypes, nil
	}

	order := make([]jira.Project, 0, len(projects))
	for _, p := range projects {
		if strings.EqualFold(p.Key, current) {
			order = append([]jira.Project{p}, order...)
		} else {
			order = append(order, p)
		}
	}
	items := make([]string, len(order))
	for i, p := range order {
		items[i] = fmt.Sprintf("%s - %s", p.Key, textutil.Truncate(p.Name, 60))
	}
	header := "Select project"
	if current != "" {
		header += " (current: " + current + ")"
	}
	idx, err := tui.Select(items, tui.SelectOptions{
		Header: header, CheckLabel: "Checking project", MaxAttempts: maxConfigureAttempts,
		Check: func(i int) error { return check(order[i].Key) },
	})
	switch {
	case tui.IsEsc(err) && current != "":
		if err := check(strings.ToUpper(current)); err != nil {
			return "", nil, err
		}
		return strings.ToUpper(current), issueTypes, nil
	case errors.Is(err, ErrCancelled):
		return "", nil, err
	case err != nil:
		return "", nil, fmt.Errorf("project check failed %d times, nothing was saved: %w", maxConfigureAttempts, err)
	}
	return order[idx].Key, issueTypes, nil
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
