package cmd

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/tui"
)

// errorHint suggests what to do about err, or "" when there's nothing useful
// to add.
func errorHint(err error) string {
	var netErr net.Error
	var urlErr *url.Error
	switch {
	case errors.Is(err, ErrNotConfigured):
		return ""
	case jira.StatusOf(err) == 401:
		return "Create a new API token at https://id.atlassian.com/manage-profile/security/api-tokens, then run 'jiractl configure' (or pick Configure in the menu)."
	case jira.StatusOf(err) == 403:
		return "Ask a Jira admin for access, or run 'jiractl doctor' to see which permissions are missing."
	case jira.StatusOf(err) == 404:
		return "Check the project key and server URL with 'jiractl doctor'."
	case errors.As(err, &netErr), errors.As(err, &urlErr):
		return "Can't reach Jira. Check your network/VPN and the server URL ('jiractl doctor' checks both)."
	case jira.StatusOf(err) != 0:
		return "Run 'jiractl doctor' to check your setup."
	}
	return ""
}

// printError shows an error and, when there is one, a hint on what to do.
func printError(title string, err error) {
	fmt.Fprintf(os.Stderr, "\n✗ %s\n  %v\n", title, err)
	if hint := errorHint(err); hint != "" {
		fmt.Fprintf(os.Stderr, "  → %s\n", hint)
	}
}

// waitForEnter pauses so a message isn't hidden by the next picker.
func waitForEnter() {
	_ = tui.Pause("Press Enter to return to the menu")
}
