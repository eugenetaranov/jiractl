package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	jiralib "github.com/andygrunwald/go-jira"
	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/textutil"
	"github.com/eugenetaranov/jiractl/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var queryCmd = &cobra.Command{
	Use:   "query [name]",
	Short: "Run a saved query or ad-hoc JQL",
	Long: `Run a saved JQL query from your config file, or any JQL with --jql.

Names match exactly, case-insensitively, or by unique prefix ("rec" runs
"recent"). Results open in a picker with a preview pane; Enter opens an
actions menu (open, copy key, transition, assign to me, comment). Esc leaves.

When stdout is not a terminal, or with -o, results are printed instead:
-o keys prints one key per line, -o json a JSON array, -o table a table.`,
	Example: `  jiractl query mine
  jiractl query --jql "project = \${project} AND labels = urgent"
  jiractl query mine -o keys | xargs -n1 echo`,
	Args: cobra.MaximumNArgs(1),
	RunE: runQueryCmd,
}

var queryOpts struct {
	jql    string
	output string
}

func init() {
	queryCmd.Flags().StringVar(&queryOpts.jql, "jql", "", "Run this JQL instead of a saved query (${project} is expanded)")
	queryCmd.Flags().StringVarP(&queryOpts.output, "output", "o", "", "Print results instead of opening the picker: keys, json or table")
	RootCmd.AddCommand(queryCmd)
}

func stdoutIsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func runQueryCmd(cmd *cobra.Command, args []string) error {
	switch queryOpts.output {
	case "", "keys", "json", "table":
	default:
		return fmt.Errorf("invalid output %q: use keys, json or table", queryOpts.output)
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if queryOpts.jql != "" {
		if len(args) > 0 {
			return errors.New("pass a query name or --jql, not both")
		}
		return executeQuery(cfg, "JQL", cfg.ExpandJQL(queryOpts.jql), 50, queryOpts.output)
	}

	var q *config.Query
	if len(args) == 0 {
		if queryOpts.output != "" || !stdoutIsTerminal() {
			return fmt.Errorf("query name required; available: %s", strings.Join(cfg.QueryNames(), ", "))
		}
		return tui.RunApp(&queryPickScreen{cfg: cfg, list: tui.NewList("queries", cfg.QueryNames(), tui.SelectOptions{Header: "Select query"})}, nil)
	} else if q, err = cfg.FindQuery(args[0]); err != nil {
		return err
	}
	return executeQuery(cfg, q.Name, cfg.ExpandJQL(q.JQL), q.Limit, queryOpts.output)
}

func executeQuery(cfg *config.Config, title, jql string, limit int, output string) error {
	client, err := jira.NewClient(cfg)
	if err != nil {
		return err
	}
	if limit <= 0 {
		limit = 50
	}

	fmt.Fprintf(os.Stderr, "Running query: %s\n", title)
	fmt.Fprintf(os.Stderr, "JQL: %s\n", jql)

	issues, err := client.SearchIssues(jql, limit)
	if err != nil {
		return fmt.Errorf("query %q failed: %w", title, err)
	}

	if output == "" && !stdoutIsTerminal() {
		output = "table"
	}
	// Lists for people put active, recently touched work first; keys and
	// JSON keep Jira's order for scripts.
	if output == "" || output == "table" {
		jira.SortForDisplay(issues)
	}
	switch output {
	case "keys":
		for _, is := range issues {
			fmt.Println(is.Key)
		}
		return nil
	case "json":
		return printIssuesJSON(issues)
	case "table":
		for _, is := range issues {
			fmt.Println(issueRow(is))
		}
		return nil
	}

	if len(issues) == 0 {
		fmt.Fprintln(os.Stderr, "No issues found.")
		return nil
	}
	return tui.RunApp(newResultsScreen(cfg, client, title, issues), nil)
}

type issueJSON struct {
	Key      string `json:"key"`
	Summary  string `json:"summary"`
	Status   string `json:"status"`
	Assignee string `json:"assignee"`
}

func printIssuesJSON(issues []jiralib.Issue) error {
	out := make([]issueJSON, 0, len(issues))
	for _, is := range issues {
		out = append(out, issueJSON{Key: is.Key, Summary: fieldSummary(is), Status: fieldStatus(is), Assignee: fieldAssignee(is)})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func fieldSummary(is jiralib.Issue) string {
	if is.Fields == nil {
		return ""
	}
	return is.Fields.Summary
}

func fieldStatus(is jiralib.Issue) string {
	if is.Fields == nil || is.Fields.Status == nil {
		return ""
	}
	return is.Fields.Status.Name
}

func fieldAssignee(is jiralib.Issue) string {
	if is.Fields == nil || is.Fields.Assignee == nil {
		return ""
	}
	return is.Fields.Assignee.DisplayName
}

func issueRow(is jiralib.Issue) string {
	return textutil.PadRight(is.Key, 12) + " " + textutil.PadRight(fieldStatus(is), 15) + " " +
		textutil.PadRight(shortAge(jira.UpdatedAt(is), time.Now()), 4) + " " + textutil.Truncate(fieldSummary(is), 60)
}

// shortAge renders how long ago t was: 5m, 3h, 2d, 3w, 4mo, 1y. Unknown
// times render as "-".
func shortAge(t, now time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 14*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 60*24*time.Hour:
		return fmt.Sprintf("%dw", int(d.Hours()/24/7))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(d.Hours()/24/30))
	}
	return fmt.Sprintf("%dy", int(d.Hours()/24/365))
}

// issuePreview renders the preview pane for an issue.
func issuePreview(is jiralib.Issue, width, height int) string {
	if is.Fields == nil {
		return is.Key
	}
	f := is.Fields
	width = max(width-2, 10)
	var lines []string
	add := func(s string) { lines = append(lines, textutil.Wrap(s, width)...) }

	add(is.Key + ": " + f.Summary)
	add(strings.Repeat("─", min(width, 40)))
	row := func(label, value string) {
		if value != "" {
			add(textutil.PadRight(label, 10) + value)
		}
	}
	row("Type", f.Type.Name)
	row("Status", fieldStatus(is))
	if f.Priority != nil {
		row("Priority", f.Priority.Name)
	}
	row("Assignee", fieldAssignee(is))
	if f.Reporter != nil {
		row("Reporter", f.Reporter.DisplayName)
	}
	if len(f.Labels) > 0 {
		row("Labels", strings.Join(f.Labels, ", "))
	}
	if updated := jira.UpdatedAt(is); !updated.IsZero() {
		row("Updated", shortAge(updated, time.Now())+" ago ("+updated.Local().Format("Jan 2 15:04")+")")
	}
	if f.Description != "" {
		add("")
		add(f.Description)
	}
	if height > 0 && len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

var issueActions = []string{"Open in browser", "Copy key", "Transition", "Assign to me", "Comment", "Back"}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
