package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/eugenetaranov/jiractl/internal/selfupdate"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version and whether a newer release exists",
	Args:  cobra.NoArgs,
	RunE:  runVersion,
}

// newUpgradeCmd builds the upgrade command; it is reachable as
// "jiractl upgrade" and "jiractl version upgrade", or "up" for short.
func newUpgradeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "upgrade",
		Aliases: []string{"up"},
		Short:   "Upgrade jiractl to the latest release",
		Long: `Upgrade jiractl to the latest GitHub release.

A Homebrew install refreshes the ` + selfupdate.Tap + ` tap, then runs
brew upgrade. Any other install downloads the release archive for this
platform, checks it against the release's checksums.txt and replaces the
running binary.`,
		Example: `  jiractl upgrade
  jiractl version upgrade --force`,
		Args: cobra.NoArgs,
		RunE: runUpgrade,
	}
	c.Flags().BoolVar(&upgradeForce, "force", false, "Reinstall even when up to date, or replace a development build")
	return c
}

var upgradeForce bool

func init() {
	versionCmd.AddCommand(newUpgradeCmd())
	RootCmd.AddCommand(versionCmd, newUpgradeCmd())
}

// upgradeError is an upgrade failure with its own hint (errorHint would
// otherwise blame Jira for network errors).
type upgradeError struct {
	err  error
	hint string
}

func (e *upgradeError) Error() string { return e.err.Error() }
func (e *upgradeError) Unwrap() error { return e.err }

func upgradeFailed(err error, hint string) error {
	return &upgradeError{err: err, hint: hint}
}

func runVersion(cmd *cobra.Command, args []string) error {
	fmt.Println(versionString())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	latest, err := (&selfupdate.Updater{}).Latest(ctx)
	switch {
	case err != nil:
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	case selfupdate.IsDev(Version):
		fmt.Printf("Latest release: %s (this is a development build)\n", latest)
	case selfupdate.Newer(Version, latest):
		fmt.Printf("Update available: %s (run 'jiractl upgrade')\n", latest)
	default:
		fmt.Println("Up to date.")
	}
	return nil
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	u := &selfupdate.Updater{}

	step("Checking the latest release")
	latest, err := u.Latest(ctx)
	if err != nil {
		return upgradeFailed(err, "Check your network, or see https://github.com/"+selfupdate.Repo+"/releases")
	}
	current := strings.TrimPrefix(Version, "v")
	if !upgradeForce {
		switch {
		case selfupdate.IsDev(Version):
			return upgradeFailed(fmt.Errorf("this is a development build (%s)", Version),
				"Pass --force to replace it with "+latest+".")
		case !selfupdate.Newer(Version, latest):
			fmt.Printf("jiractl %s is up to date.\n", current)
			return nil
		}
	}

	exe, err := selfupdate.Executable()
	if err != nil {
		return upgradeFailed(fmt.Errorf("could not find the running binary: %w", err), "")
	}
	if selfupdate.IsBrew(exe) {
		err = brewUpgrade(latest)
	} else {
		step("Installing " + latest + " to " + exe)
		err = u.Install(ctx, latest, exe)
		if err != nil {
			err = upgradeFailed(err, "")
		}
	}
	if err != nil {
		return err
	}
	fmt.Printf("Upgraded jiractl %s → %s\n", current, strings.TrimPrefix(latest, "v"))
	return nil
}

// brewUpgrade refreshes the tap (Homebrew's cached copy of the formula) and
// upgrades the formula to latest.
func brewUpgrade(latest string) error {
	if _, err := exec.LookPath("brew"); err != nil {
		return upgradeFailed(errors.New("jiractl was installed with Homebrew, but brew is not on PATH"), "")
	}
	formula := selfupdate.Tap + "/" + selfupdate.Formula

	step("Refreshing the " + selfupdate.Tap + " tap")
	if err := refreshTap(); err != nil {
		return upgradeFailed(fmt.Errorf("could not refresh the tap: %w", err), "Try 'brew update', then 'jiractl upgrade' again.")
	}

	tapVersion, err := brewFormulaVersion(formula)
	if err != nil {
		return upgradeFailed(err, "")
	}
	if selfupdate.Newer(tapVersion, latest) {
		return upgradeFailed(fmt.Errorf("the tap still has %s; %s is not in Homebrew yet", tapVersion, latest),
			"The formula is published a few minutes after the release. Try again shortly.")
	}

	step("Running brew upgrade " + formula)
	upgrade := brew("upgrade", formula)
	upgrade.Env = append(os.Environ(), "HOMEBREW_NO_AUTO_UPDATE=1")
	if err := upgrade.Run(); err != nil {
		return upgradeFailed(fmt.Errorf("brew upgrade failed: %w", err), "")
	}

	prefix, err := brewOutput("--prefix")
	if err != nil {
		return nil // installed; just can't verify
	}
	bin := filepath.Join(prefix, "bin", selfupdate.Binary)
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return nil
	}
	// "jiractl <version> (commit: ..., built: ...)"
	if fields := strings.Fields(string(out)); len(fields) > 1 && selfupdate.Newer(fields[1], latest) {
		return upgradeFailed(fmt.Errorf("%s is still %s after brew upgrade", bin, fields[1]), "")
	}
	return nil
}

// refreshTap pulls the tap's git checkout, tapping it when it's missing.
// Pulling one tap is much faster than a full brew update.
func refreshTap() error {
	dir, err := brewOutput("--repository", selfupdate.Tap)
	if err == nil {
		if _, statErr := os.Stat(filepath.Join(dir, ".git")); statErr == nil {
			pull := exec.Command("git", "-C", dir, "pull", "--ff-only", "-q")
			pull.Stdout, pull.Stderr = os.Stderr, os.Stderr
			if pull.Run() == nil {
				return nil
			}
			return brew("update").Run()
		}
	}
	return brew("tap", selfupdate.Tap).Run()
}

func brewFormulaVersion(formula string) (string, error) {
	out, err := exec.Command("brew", "info", "--json=v2", formula).Output()
	if err != nil {
		return "", fmt.Errorf("brew info %s failed: %w", formula, err)
	}
	var info struct {
		Formulae []struct {
			Versions struct {
				Stable string `json:"stable"`
			} `json:"versions"`
		} `json:"formulae"`
	}
	if err := json.Unmarshal(out, &info); err != nil || len(info.Formulae) == 0 {
		return "", fmt.Errorf("could not read brew info for %s", formula)
	}
	return info.Formulae[0].Versions.Stable, nil
}

// brew runs brew with its output on stderr, so stdout keeps only results.
func brew(args ...string) *exec.Cmd {
	c := exec.Command("brew", args...)
	c.Stdout, c.Stderr = os.Stderr, os.Stderr
	return c
}

func brewOutput(args ...string) (string, error) {
	out, err := exec.Command("brew", args...).Output()
	return strings.TrimSpace(string(out)), err
}

func step(msg string) {
	fmt.Fprintln(os.Stderr, "==> "+msg)
}
