package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// EditorCommand returns the user's editor: $EDITOR, then $VISUAL, then vi
// (notepad on Windows). The value may include arguments, e.g. "code -w".
func EditorCommand() []string {
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

// editorSession is a temp file holding text being edited.
type editorSession struct {
	path string
	cmd  *exec.Cmd
}

func newEditorSession(initial string) (*editorSession, error) {
	f, err := os.CreateTemp("", "jiractl-*.md")
	if err != nil {
		return nil, err
	}
	if initial != "" {
		initial += "\n"
	}
	_, err = f.WriteString(initial)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(f.Name())
		return nil, err
	}
	args := EditorCommand()
	cmd := exec.Command(args[0], append(args[1:], f.Name())...)
	return &editorSession{path: f.Name(), cmd: cmd}, nil
}

// result reads the edited text back and removes the temp file.
func (s *editorSession) result(runErr error) (string, error) {
	defer func() { _ = os.Remove(s.path) }()
	if runErr != nil {
		return "", fmt.Errorf("editor %q failed: %w", strings.Join(s.cmd.Args[:len(s.cmd.Args)-1], " "), runErr)
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\r\n"), nil
}

// EditText opens text in the user's editor outside any program and returns
// the result without trailing newlines.
func EditText(initial string) (string, error) {
	s, err := newEditorSession(initial)
	if err != nil {
		return "", err
	}
	s.cmd.Stdin, s.cmd.Stdout, s.cmd.Stderr = os.Stdin, os.Stderr, os.Stderr
	return s.result(s.cmd.Run())
}
