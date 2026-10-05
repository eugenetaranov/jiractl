package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// editorCommand returns the user's editor: $EDITOR, then $VISUAL, then vi
// (notepad on Windows). The value may include arguments, e.g. "code -w".
func editorCommand() []string {
	for _, env := range []string{"EDITOR", "VISUAL"} {
		if v := strings.Fields(os.Getenv(env)); len(v) > 0 {
			return v
		}
	}
	if runtime.GOOS == "windows" {
		return []string{"notepad"}
	}
	return []string{"vi"}
}

// openEditor lets the user edit text in their editor and returns the result
// without trailing newlines.
func openEditor(initial string) (string, error) {
	f, err := os.CreateTemp("", "jiractl-*.md")
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer func() { _ = os.Remove(path) }()

	if initial != "" {
		initial += "\n"
	}
	_, err = f.WriteString(initial)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", err
	}

	args := editorCommand()
	cmd := exec.Command(args[0], append(args[1:], path)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor %q failed: %w", strings.Join(args, " "), err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\r\n"), nil
}
