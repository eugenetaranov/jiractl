package cmd

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/keyring"
	"github.com/manifoldco/promptui"
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
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current credentials for defaults
	currentUsername, _ := keyring.GetUsername()

	// Prompt for server URL
	serverPrompt := promptui.Prompt{
		Label:   "Jira Server URL",
		Default: cfg.Server,
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("server URL is required")
			}
			if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
				return fmt.Errorf("server URL must start with http:// or https://")
			}
			return nil
		},
	}
	server, err := serverPrompt.Run()
	if err != nil {
		return handlePromptError(err)
	}
	cfg.Server = strings.TrimRight(server, "/")

	// Prompt for project key
	projectPrompt := promptui.Prompt{
		Label:   "Default Project Key",
		Default: cfg.Project,
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("project key is required")
			}
			return nil
		},
	}
	project, err := projectPrompt.Run()
	if err != nil {
		return handlePromptError(err)
	}
	cfg.Project = strings.ToUpper(project)

	// Prompt for username
	usernamePrompt := promptui.Prompt{
		Label:   "Username (email)",
		Default: currentUsername,
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("username is required")
			}
			return nil
		},
	}
	username, err := usernamePrompt.Run()
	if err != nil {
		return handlePromptError(err)
	}

	// Prompt for API token using term.ReadPassword (handles paste correctly)
	fmt.Print("API Token: ")
	tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		return fmt.Errorf("API token is required")
	}

	// Save credentials to keyring
	if err := keyring.SetUsername(username); err != nil {
		return fmt.Errorf("failed to save username: %w", err)
	}
	if err := keyring.SetToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	// Save config to file
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("\nConfiguration saved!")
	fmt.Printf("  Config file: ~/.jiractl.toml\n")
	fmt.Printf("  Credentials: stored in system keyring\n")

	// Test connection
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
	return nil
}

func handlePromptError(err error) error {
	if err == promptui.ErrInterrupt || err == promptui.ErrEOF {
		fmt.Println("\nConfiguration cancelled.")
		return nil
	}
	return err
}
