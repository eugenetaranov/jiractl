package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

var (
	debug       bool
	showVersion bool
)

// errVersionShown stops command execution after --version printed the version.
var errVersionShown = errors.New("version shown")

var RootCmd = &cobra.Command{
	Use:           "jiractl",
	Short:         "CLI tool for interacting with Jira",
	Long:          `jiractl is a command-line interface for managing Jira issues, projects, and workflows.`,
	RunE:          runInteractiveMenu,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(versionString())
			return errVersionShown
		}
		return nil
	},
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output")
}

func versionString() string {
	return fmt.Sprintf("jiractl %s (commit: %s, built: %s)", Version, Commit, Date)
}

func runInteractiveMenu(cmd *cobra.Command, args []string) error {
	menuItems := []string{
		"Create new issue",
		"Run query",
		"Configure",
		"Exit",
	}

	idx, err := fzfSelect(menuItems, "Select action")
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return nil
		}
		return fmt.Errorf("prompt failed: %w", err)
	}

	switch idx {
	case 0: // Create new issue
		return createCmd.RunE(createCmd, nil)
	case 1: // Run query
		return runQueryInteractive()
	case 2: // Configure
		return configureCmd.RunE(configureCmd, nil)
	case 3: // Exit
		return nil
	}

	return nil
}

func runQueryInteractive() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if len(cfg.Queries) == 0 {
		fmt.Fprintln(os.Stderr, "No queries configured. Add queries to ~/.jiractl.toml")
		return nil
	}

	names := cfg.QueryNames()
	idx, err := fzfSelect(names, "Select query")
	if err != nil {
		return err
	}

	return runQuery(names[idx])
}

func Execute() {
	err := RootCmd.Execute()
	switch {
	case err == nil, errors.Is(err, errVersionShown):
		return
	case errors.Is(err, ErrCancelled):
		fmt.Fprintln(os.Stderr, "Cancelled.")
		os.Exit(130)
	default:
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
