package cmd

import (
	"fmt"
	"os"

	"github.com/eugenetaranov/jiractl/internal/doctor"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check that jiractl is ready to create issues and run queries",
	Long: `Check the config file, credentials, server, authentication, project,
permissions, configured defaults and saved queries, and say what to fix.

Exits 1 when any check fails; warnings (things that only disable one
action) don't change the exit status.`,
	Args: cobra.NoArgs,
	RunE: runDoctor,
}

var doctorJSON bool

func init() {
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Print results as JSON")
	RootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	results := doctor.Run(doctor.NewCtx(), doctor.Checks(true))
	if doctorJSON {
		if err := doctor.PrintJSON(os.Stdout, results); err != nil {
			return err
		}
	} else if err := doctor.Print(os.Stdout, results, fancyOutput()); err != nil {
		return err
	}
	if doctor.Failed(results) {
		return errSilentFailure
	}
	return nil
}

// fancyOutput reports whether ✓/✗ markers and color can be used.
func fancyOutput() bool {
	return term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("NO_COLOR") == ""
}

// runPostConfigureDoctor shows what still needs fixing after configure.
func runPostConfigureDoctor() {
	fmt.Fprintln(os.Stderr, "\nChecking setup...")
	results := doctor.Run(doctor.NewCtx(), doctor.Checks(false))
	_ = doctor.Print(os.Stderr, results, term.IsTerminal(int(os.Stderr.Fd())) && os.Getenv("NO_COLOR") == "")
}
