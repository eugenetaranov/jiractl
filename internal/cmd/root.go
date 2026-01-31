package cmd

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
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

var RootCmd = &cobra.Command{
	Use:   "jiractl",
	Short: "CLI tool for interacting with Jira",
	Long:  `jiractl is a command-line interface for managing Jira issues, projects, and workflows.`,
	RunE:  runInteractiveMenu,
}

func init() {
	RootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output")
}

func runInteractiveMenu(cmd *cobra.Command, args []string) error {
	if showVersion {
		fmt.Printf("jiractl %s (commit: %s, built: %s)\n", Version, Commit, Date)
		return nil
	}

	menuItems := []string{
		"Create new issue",
		"Run query",
		"Configure",
		"Exit",
	}

	prompt := promptui.Select{
		Label: "What would you like to do?",
		Items: menuItems,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
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
		fmt.Println("No queries configured. Add queries to ~/.jiractl.toml")
		return nil
	}

	names := cfg.QueryNames()
	prompt := promptui.Select{
		Label: "Select a query",
		Items: names,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			return nil
		}
		return fmt.Errorf("prompt failed: %w", err)
	}

	return runQuery(names[idx])
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
