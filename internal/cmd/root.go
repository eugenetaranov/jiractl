package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
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

// errSilentFailure exits 1 for a command that already reported its problems.
var errSilentFailure = errors.New("failed")

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

// ErrNotConfigured is returned when there is no server or project and setup
// can't be offered (no terminal) or was declined.
var ErrNotConfigured = errors.New("not configured: run 'jiractl configure'")

// menuStatus is a one-line result of the last menu action (e.g. the created
// key), shown in the menu header because the picker hides earlier output.
var menuStatus string

// loadConfig returns the config, offering to run setup when it is missing.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if cfg.Server != "" && cfg.Project != "" {
		return cfg, nil
	}
	if !isInteractive() || !stdoutIsTerminal() {
		return nil, ErrNotConfigured
	}

	setup, err := promptConfirm("jiractl isn't set up yet. Run setup now?", true)
	if err != nil {
		return nil, err
	}
	if !setup {
		return nil, ErrNotConfigured
	}
	if err := runConfigure(configureCmd, nil); err != nil {
		return nil, err
	}
	if cfg, err = config.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if cfg.Server == "" || cfg.Project == "" {
		return nil, ErrNotConfigured
	}
	return cfg, nil
}

var menuItems = []string{
	"Create new issue",
	"Run query",
	"Change default epic",
	"Configure",
	"Exit",
}

// runInteractiveMenu shows the main menu until Exit or Esc. Errors and
// cancels inside an action are reported and the menu comes back.
func runInteractiveMenu(cmd *cobra.Command, args []string) error {
	if _, err := loadConfig(); err != nil {
		return err
	}

	authWarning := checkLogin()
	for {
		header := "Select action"
		// Reload each time: configure or the epic entry may have changed it.
		if cfg, err := config.Load(); err == nil {
			client, _ := jira.NewClient(cfg)
			header = defaultEpicLine(cfg, client) + "  │  " + header
		}
		if authWarning != "" {
			header = "⚠ " + authWarning + "  │  " + header
		}
		if menuStatus != "" {
			header += "   (" + menuStatus + ")"
		}
		idx, err := fzfSelect(menuItems, header)
		if errors.Is(err, ErrCancelled) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("prompt failed: %w", err)
		}

		menuStatus = ""
		action := menuItems[idx]
		var actionErr error
		switch menuItems[idx] {
		case "Create new issue":
			actionErr = createCmd.RunE(createCmd, nil)
		case "Run query":
			actionErr = runQueryInteractive()
		case "Change default epic":
			actionErr = changeDefaultEpic()
		case "Configure":
			actionErr = configureCmd.RunE(configureCmd, nil)
			if actionErr == nil {
				authWarning = checkLogin()
			}
		case "Exit":
			return nil
		}

		switch {
		case actionErr == nil:
		case errors.Is(actionErr, ErrCancelled):
			fmt.Fprintln(os.Stderr, "Cancelled.")
			menuStatus = "Cancelled."
		default:
			printError(action+" failed", actionErr)
			waitForEnter()
			menuStatus = action + " failed"
		}
	}
}

// checkLogin tests the stored credentials so the menu can say up front that
// Jira rejects them. It returns "" when login works or can't be checked.
func checkLogin() string {
	cfg, err := config.Load()
	if err != nil {
		return ""
	}
	client, err := jira.NewClient(cfg)
	if err != nil {
		return "No credentials stored: choose Configure"
	}
	p := fetch(func() (struct{}, error) { return struct{}{}, client.TestConnection() })
	_, err, done := p.waitFor(3 * time.Second)
	if !done || err == nil {
		return ""
	}
	if s := jira.StatusOf(err); s == 401 || s == 403 {
		return "Jira rejected your API token (expired?): choose Configure"
	}
	return "Can't reach Jira: run 'jiractl doctor'"
}

func Execute() {
	err := RootCmd.Execute()
	switch {
	case err == nil, errors.Is(err, errVersionShown):
		return
	case errors.Is(err, ErrCancelled):
		fmt.Fprintln(os.Stderr, "Cancelled.")
		os.Exit(130)
	case errors.Is(err, errSilentFailure):
		os.Exit(1)
	default:
		fmt.Fprintln(os.Stderr, "Error:", err)
		if hint := errorHint(err); hint != "" {
			fmt.Fprintln(os.Stderr, "  →", hint)
		}
		os.Exit(1)
	}
}
