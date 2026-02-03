package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
)

var ErrPromptCancelled = errors.New("cancelled")

// fzfSelect provides a selection UI with a header showing the prompt
func fzfSelect(items []string, prompt ...string) (int, error) {
	opts := []fuzzyfinder.Option{}
	if len(prompt) > 0 && prompt[0] != "" {
		opts = append(opts, fuzzyfinder.WithHeader(prompt[0]))
	}
	return fuzzyfinder.Find(items, func(i int) string {
		return items[i]
	}, opts...)
}

// promptText prompts for text input with readline support (Ctrl+W, etc.)
func promptText(label string, required bool) (string, error) {
	return promptTextWithDefault(label, "", required)
}

// promptTextWithDefault prompts for text input with a default value
func promptTextWithDefault(label, defaultVal string, required bool) (string, error) {
	prompt := label + ": "
	if defaultVal != "" {
		prompt = label + " [" + defaultVal + "]: "
	}

	rl, err := readline.New(prompt)
	if err != nil {
		return "", err
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt || err == io.EOF {
			return "", ErrPromptCancelled
		}
		if err != nil {
			return "", err
		}

		line = strings.TrimSpace(line)
		if line == "" && defaultVal != "" {
			return defaultVal, nil
		}
		if required && line == "" {
			fmt.Println("This field is required")
			continue
		}
		return line, nil
	}
}

// promptMultilineText prompts for multiline text input. Empty line signals completion.
func promptMultilineText(label string) (string, error) {
	fmt.Printf("%s (empty line to finish):\n", label)

	rl, err := readline.New("> ")
	if err != nil {
		return "", err
	}
	defer rl.Close()

	var lines []string
	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt || err == io.EOF {
			return "", ErrPromptCancelled
		}
		if err != nil {
			return "", err
		}

		if line == "" {
			break
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n"), nil
}

// promptConfirm prompts for y/n confirmation
func promptConfirm(label string) (bool, error) {
	rl, err := readline.New(label + " [y/N]: ")
	if err != nil {
		return false, err
	}
	defer rl.Close()

	line, err := rl.Readline()
	if err == readline.ErrInterrupt || err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes", nil
}

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

	idx, err := fzfSelect(menuItems, "Select action")
	if err != nil {
		if err == fuzzyfinder.ErrAbort {
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
	idx, err := fzfSelect(names, "Select query")
	if err != nil {
		if err == fuzzyfinder.ErrAbort {
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
