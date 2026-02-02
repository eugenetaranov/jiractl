package cmd

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/keyring"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication credentials",
	Long:  `Manage Jira authentication credentials stored in the system keyring.`,
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored credentials",
	RunE:  runAuthList,
}

var authDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete stored credentials",
	RunE:  runAuthDelete,
}

var authCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create/update credentials",
	RunE:  runAuthCreate,
}

var authTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test connection with stored credentials",
	RunE:  runAuthTest,
}

func init() {
	RootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authDeleteCmd)
	authCmd.AddCommand(authCreateCmd)
	authCmd.AddCommand(authTestCmd)
}

func runAuthList(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	username, err := keyring.GetUsername()
	if err != nil {
		return fmt.Errorf("failed to get username: %w", err)
	}

	token, err := keyring.GetToken()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	if username == "" && token == "" {
		fmt.Println("No credentials stored.")
		return nil
	}

	fmt.Println("Stored credentials:")
	if username != "" {
		fmt.Printf("  Username: %s\n", username)
	} else {
		fmt.Println("  Username: (not set)")
	}

	if token != "" {
		// Show masked token
		masked := token[:4] + "..." + token[len(token)-4:]
		if len(token) < 12 {
			masked = "****"
		}
		fmt.Printf("  Token:    %s\n", masked)
	} else {
		fmt.Println("  Token:    (not set)")
	}

	return nil
}

func runAuthDelete(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	if !keyring.HasCredentials() {
		fmt.Println("No credentials stored.")
		return nil
	}

	confirmed, err := promptConfirm("Delete stored credentials?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := keyring.ClearCredentials(); err != nil {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	fmt.Println("Credentials deleted.")
	return nil
}

func runAuthCreate(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	currentUsername, _ := keyring.GetUsername()

	// Prompt for username
	username, err := promptTextWithDefault("Username (email)", currentUsername, true)
	if err != nil {
		if err == ErrPromptCancelled {
			fmt.Println("\nCancelled.")
			return nil
		}
		return err
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

	// Save credentials
	if err := keyring.SetUsername(username); err != nil {
		return fmt.Errorf("failed to save username: %w", err)
	}
	if err := keyring.SetToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("Credentials saved to system keyring.")
	return nil
}

func runAuthTest(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	if !keyring.HasCredentials() {
		return fmt.Errorf("no credentials stored, run 'jiractl auth create' first")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Server == "" {
		return fmt.Errorf("server not configured, run 'jiractl configure' first")
	}

	username, token, _ := keyring.GetCredentials()

	fmt.Printf("Testing connection to %s...\n", cfg.Server)
	if debug {
		fmt.Printf("  Username: %s\n", username)
		fmt.Printf("  Token length: %d\n", len(token))
		fmt.Printf("  Token prefix: %s\n", token[:min(8, len(token))])
	}

	client, err := jira.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if err := client.TestConnection(); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	fmt.Println("Connection successful!")
	return nil
}
