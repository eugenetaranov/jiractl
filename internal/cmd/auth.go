package cmd

import (
	"fmt"
	"os"

	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/doctor"
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

// runAuthCreate is the credentials step of configure on its own: same
// prompts, same checks, nothing saved until Jira accepts the token.
func runAuthCreate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	serverChanged := false
	if cfg.Server == "" {
		if cfg.Server, err = askServer(""); err != nil {
			return err
		}
		serverChanged = true
	}

	creds, err := promptCredentials(cfg)
	if err != nil {
		return err
	}
	if err := creds.save(); err != nil {
		return err
	}
	if serverChanged {
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	fmt.Println("Credentials saved to system keyring.")
	return nil
}

// runAuthTest runs the credential-related doctor checks.
func runAuthTest(cmd *cobra.Command, args []string) error {
	var checks []doctor.Check
	for _, c := range doctor.Checks(true) {
		switch c.ID {
		case "config", "creds", "server", "auth":
			checks = append(checks, c)
		}
	}
	ctx := doctor.NewCtx()
	if debug {
		fmt.Fprintf(os.Stderr, "  Username: %s\n", ctx.Username)
		fmt.Fprintf(os.Stderr, "  Token length: %d\n", len(ctx.Token))
	}
	results := doctor.Run(ctx, checks)
	doctor.Print(os.Stdout, results, fancyOutput())
	if doctor.Failed(results) {
		return errSilentFailure
	}
	return nil
}
