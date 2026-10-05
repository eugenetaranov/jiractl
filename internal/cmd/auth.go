package cmd

import (
	"fmt"
	"os"

	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/keyring"
	"github.com/spf13/cobra"
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
		fmt.Printf("  Token:    set (%d chars)\n", len(token))
	} else {
		fmt.Println("  Token:    (not set)")
	}

	return nil
}

func runAuthDelete(cmd *cobra.Command, args []string) error {
	if !keyring.HasCredentials() {
		fmt.Println("No credentials stored.")
		return nil
	}

	confirmed, err := promptConfirm("Delete stored credentials?", false)
	if err != nil {
		return err
	}
	if !confirmed {
		return ErrCancelled
	}

	if err := keyring.ClearCredentials(); err != nil {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	fmt.Println("Credentials deleted.")
	return nil
}

func runAuthCreate(cmd *cobra.Command, args []string) error {
	currentUsername, _ := keyring.GetUsername()

	// Prompt for username
	username, err := promptTextWithDefault("Username (email)", currentUsername, true)
	if err != nil {
		return err
	}

	token, err := readSecret("API Token: ")
	if err != nil {
		return err
	}
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

	fmt.Fprintf(os.Stderr, "Testing connection to %s...\n", cfg.Server)
	if debug {
		fmt.Fprintf(os.Stderr, "  Username: %s\n", username)
		fmt.Fprintf(os.Stderr, "  Token length: %d\n", len(token))
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
