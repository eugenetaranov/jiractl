package cmd

import "github.com/eugenetaranov/jiractl/internal/tui"

// openEditor lets the user edit text in their editor and returns the result
// without trailing newlines.
func openEditor(initial string) (string, error) {
	return tui.EditText(initial)
}
