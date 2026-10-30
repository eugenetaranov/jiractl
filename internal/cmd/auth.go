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
	Long: `Manage Jira authentication credentials.

Credentials are read from, in order: the JIRACTL_USERNAME and JIRACTL_TOKEN
environment variables, the system keyring, and ~/.config/jiractl/credentials
(mode 0600). They are saved to the system keyring when it is available and to
the credentials file otherwise, e.g. on Linux without a desktop session.
Set JIRACTL_CREDENTIAL_STORE=keyring or =file to pick one.`,
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

	usernameSource, tokenSource := keyring.Sources()
	fmt.Println("Stored credentials:")
	if username != "" {
		fmt.Printf("  Username: %s (from %s)\n", username, sourceName(usernameSource))
	} else {
		fmt.Println("  Username: (not set)")
	}

	if token != "" {
		fmt.Printf("  Token:    set (%d chars, from %s)\n", len(token), sourceName(tokenSource))
	} else {
		fmt.Println("  Token:    (not set)")
	}

	return nil
}

// sourceName describes a credential source for output.
func sourceName(s keyring.Source) string {
	switch s {
	case keyring.SourceEnv:
		return "environment"
	case keyring.SourceKeyring:
		return "system keyring"
	case keyring.SourceFile:
		return keyring.DisplayPath()
	}
	return string(s)
}

func runAuthDelete(cmd *cobra.Command, args []string) error {
	usernameSource, tokenSource := keyring.Sources()
	stored := func(s keyring.Source) bool { return s == keyring.SourceKeyring || s == keyring.SourceFile }
	if !stored(usernameSource) && !stored(tokenSource) {
		fmt.Println("No credentials stored.")
		warnEnvCredentials()
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
	warnEnvCredentials()
	return nil
}

// warnEnvCredentials notes credentials that come from the environment,
// which auth delete cannot remove.
func warnEnvCredentials() {
	if os.Getenv(keyring.UsernameEnv) != "" || os.Getenv(keyring.TokenEnv) != "" {
		fmt.Fprintf(os.Stderr, "Note: %s/%s are set in the environment and still apply; unset them to remove.\n", keyring.UsernameEnv, keyring.TokenEnv)
	}
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
		if cfg.Server, cfg.Deployment, err = askServer(""); err != nil {
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

	fmt.Printf("Credentials saved to %s.\n", keyring.WriteLocation())
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
	if err := doctor.Print(os.Stdout, results, fancyOutput()); err != nil {
		return err
	}
	if doctor.Failed(results) {
		return errSilentFailure
	}
	return nil
}
