package cmd

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/keyring"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure jiractl settings",
	Long:  `Interactive setup for jiractl. Prompts for server URL, project key, username, and API token.`,
	RunE:  runConfigure,
}

func init() {
	RootCmd.AddCommand(configureCmd)
}

func runConfigure(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current credentials for defaults
	currentUsername, _ := keyring.GetUsername()

	// Prompt for server URL
	server, err := promptTextWithDefault("Jira Server URL", cfg.Server, true)
	if err != nil {
		if err == ErrPromptCancelled {
			fmt.Println("\nConfiguration cancelled.")
			return nil
		}
		return err
	}
	if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
		return fmt.Errorf("server URL must start with http:// or https://")
	}
	cfg.Server = strings.TrimRight(server, "/")

	// Prompt for project key
	project, err := promptTextWithDefault("Default Project Key", cfg.Project, true)
	if err != nil {
		if err == ErrPromptCancelled {
			fmt.Println("\nConfiguration cancelled.")
			return nil
		}
		return err
	}
	cfg.Project = strings.ToUpper(project)

	// Prompt for username
	username, err := promptTextWithDefault("Username (email)", currentUsername, true)
	if err != nil {
		if err == ErrPromptCancelled {
			fmt.Println("\nConfiguration cancelled.")
			return nil
		}
		return err
	}

	// Check if token already exists
	existingToken, _ := keyring.GetToken()
	hasExistingToken := existingToken != ""

	// Prompt for API token using term.ReadPassword (handles paste correctly)
	if hasExistingToken {
		fmt.Print("API Token (leave empty to keep existing): ")
	} else {
		fmt.Print("API Token: ")
	}
	tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	// Require token on first-time setup
	if token == "" && !hasExistingToken {
		return fmt.Errorf("API token is required")
	}

	// Save credentials to keyring
	credentialsUpdated := false
	if err := keyring.SetUsername(username); err != nil {
		return fmt.Errorf("failed to save username: %w", err)
	}
	// Only update token if a new one was entered
	if token != "" {
		if err := keyring.SetToken(token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}
		credentialsUpdated = true
	}

	// Save config to file (initial save to test connection)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Test connection and fetch issue types
	fmt.Print("\nTesting connection... ")
	client, err := jira.NewClient(cfg)
	if err != nil {
		fmt.Printf("failed: %v\n", err)
		return nil
	}

	if err := client.TestConnection(); err != nil {
		fmt.Printf("failed: %v\n", err)
		return nil
	}
	fmt.Println("success!")

	// Fetch issue types and prompt for default
	issueTypes, err := client.GetIssueTypes(cfg.Project)
	if err != nil {
		fmt.Printf("Warning: could not fetch issue types: %v\n", err)
	} else if len(issueTypes) > 0 {
		typeNames := make([]string, len(issueTypes))
		for i, it := range issueTypes {
			typeNames[i] = it.Name
		}

		idx, err := fzfSelect(typeNames, "Select default issue type")
		if err != nil {
			if err != fuzzyfinder.ErrAbort {
				return err
			}
		} else {
			cfg.IssueDefaults.IssueType = typeNames[idx]
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
		}
	}

	// Fetch epics and prompt for default epic link
	epics, err := client.GetEpics(cfg.Project)
	if err != nil {
		fmt.Printf("Warning: could not fetch epics: %v\n", err)
	} else if len(epics) > 0 {
		epicItems := make([]string, len(epics)+1)
		epicItems[0] = "(None)"
		for i, epic := range epics {
			summary := ""
			if epic.Fields != nil {
				summary = epic.Fields.Summary
			}
			if len(summary) > 50 {
				summary = summary[:47] + "..."
			}
			epicItems[i+1] = fmt.Sprintf("%s - %s", epic.Key, summary)
		}

		idx, err := fzfSelect(epicItems, "Select default epic (optional)")
		if err != nil {
			if err != fuzzyfinder.ErrAbort {
				return err
			}
		} else if idx > 0 {
			// User selected an epic (not "None")
			cfg.IssueDefaults.EpicLink = epics[idx-1].Key
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
		} else {
			// User selected "None", clear any existing epic link
			cfg.IssueDefaults.EpicLink = ""
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
		}
	}

	fmt.Println("\nConfiguration saved!")
	fmt.Printf("  Config file: ~/.jiractl.toml\n")
	if credentialsUpdated {
		fmt.Printf("  Credentials: stored in system keyring\n")
	}

	return nil
}
