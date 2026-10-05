package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/eugenetaranov/jiractl/internal/tui"
)

// ErrCancelled is returned by every prompt and picker when the user presses
// Esc or Ctrl+C. Execute turns it into "Cancelled." and exit 130. Callers
// that treat Esc as "skip" check tui.IsEsc, so Ctrl+C still stops jiractl.
var ErrCancelled = tui.ErrCancelled

// isInteractive reports whether prompts can be shown.
func isInteractive() bool {
	return tui.IsTerminal()
}

// fzfSelect shows an inline, filterable list with an optional header.
func fzfSelect(items []string, prompt ...string) (int, error) {
	opts := tui.SelectOptions{}
	if len(prompt) > 0 {
		opts.Header = prompt[0]
	}
	return tui.Select(items, opts)
}

// promptText prompts for one line of text.
func promptText(label string, required bool) (string, error) {
	return promptTextWithDefault(label, "", required)
}

// promptTextWithDefault prompts for one line of text; Enter on an empty
// field returns defaultVal.
func promptTextWithDefault(label, defaultVal string, required bool) (string, error) {
	return tui.Input(label, tui.InputOptions{Default: defaultVal, Required: required})
}

// promptMultilineText reads text that may contain blank lines: Enter adds a
// line, Ctrl+D finishes, Ctrl+E continues in $EDITOR.
func promptMultilineText(label string) (string, error) {
	return tui.Textarea(label, "")
}

// promptConfirm asks a yes/no question. Enter picks the default.
func promptConfirm(label string, defaultYes bool) (bool, error) {
	return tui.Confirm(label, defaultYes)
}

// readSecret reads a value without echo. Without a terminal (e.g. piped
// input) it reads one line from stdin.
func readSecret(label string) (string, error) {
	if !tui.IsTerminal() {
		fmt.Fprint(os.Stderr, label)
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return "", ErrCancelled
		}
		return strings.TrimSpace(line), nil
	}
	return tui.Secret(strings.TrimSuffix(strings.TrimSpace(label), ":"), tui.InputOptions{})
}
