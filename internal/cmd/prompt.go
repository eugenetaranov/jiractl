package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/chzyer/readline"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"golang.org/x/term"
)

// ErrCancelled is returned by every prompt and picker when the user presses
// Ctrl+C, Ctrl+D or Esc. Execute turns it into "Cancelled." and exit 130.
var ErrCancelled = errors.New("cancelled")

// fzfSelect provides a selection UI with a header showing the prompt.
// Esc returns ErrCancelled; callers for optional pickers treat that as "skip".
func fzfSelect(items []string, prompt ...string) (int, error) {
	opts := []fuzzyfinder.Option{}
	if len(prompt) > 0 && prompt[0] != "" {
		opts = append(opts, fuzzyfinder.WithHeader(prompt[0]))
	}
	idx, err := fuzzyfinder.Find(items, func(i int) string {
		return items[i]
	}, opts...)
	if errors.Is(err, fuzzyfinder.ErrAbort) {
		return 0, ErrCancelled
	}
	return idx, err
}

func newReadline(prompt string) (*readline.Instance, error) {
	// Prompts go to stderr so stdout only carries command results.
	return readline.NewEx(&readline.Config{Prompt: prompt, Stdout: os.Stderr, Stderr: os.Stderr})
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

	rl, err := newReadline(prompt)
	if err != nil {
		return "", err
	}
	defer func() { _ = rl.Close() }()

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt || err == io.EOF {
			return "", ErrCancelled
		}
		if err != nil {
			return "", err
		}

		line = strings.TrimSpace(line)
		if line == "" && defaultVal != "" {
			return defaultVal, nil
		}
		if required && line == "" {
			fmt.Fprintln(os.Stderr, "This field is required")
			continue
		}
		return line, nil
	}
}

// promptMultilineText reads text that may contain blank lines. A line with
// only "." or Ctrl+D finishes; ":e" on its own line opens $EDITOR with what
// was typed so far. Ctrl+C cancels.
func promptMultilineText(label string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s (finish with \".\" on its own line or Ctrl+D; \":e\" opens your editor):\n", label)

	rl, err := newReadline("> ")
	if err != nil {
		return "", err
	}
	defer func() { _ = rl.Close() }()

	var lines []string
	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			return "", ErrCancelled
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch strings.TrimSpace(line) {
		case ".":
			return joinLines(lines), nil
		case ":e":
			_ = rl.Close()
			return openEditor(joinLines(lines))
		}
		lines = append(lines, line)
	}

	return joinLines(lines), nil
}

// joinLines joins lines and drops trailing blank ones.
func joinLines(lines []string) string {
	return strings.TrimRight(strings.Join(lines, "\n"), "\n ")
}

// promptConfirm asks a yes/no question. Enter picks the default.
func promptConfirm(label string, defaultYes bool) (bool, error) {
	hint := " [y/N]: "
	if defaultYes {
		hint = " [Y/n]: "
	}
	rl, err := newReadline(label + hint)
	if err != nil {
		return false, err
	}
	defer func() { _ = rl.Close() }()

	line, err := rl.Readline()
	if err == readline.ErrInterrupt || err == io.EOF {
		return false, ErrCancelled
	}
	if err != nil {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(line)) {
	case "":
		return defaultYes, nil
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// readSecret reads a value without echo. Ctrl+C restores the terminal before
// returning ErrCancelled, so the shell never stays in no-echo mode.
func readSecret(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	fd := int(os.Stdin.Fd())

	if !term.IsTerminal(fd) {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return "", ErrCancelled
		}
		return strings.TrimSpace(line), nil
	}

	state, err := term.GetState(fd)
	if err != nil {
		return "", err
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	type result struct {
		b   []byte
		err error
	}
	done := make(chan result, 1)
	go func() {
		b, err := term.ReadPassword(fd)
		done <- result{b, err}
	}()

	select {
	case r := <-done:
		fmt.Fprintln(os.Stderr)
		if r.err != nil {
			return "", fmt.Errorf("failed to read input: %w", r.err)
		}
		return strings.TrimSpace(string(r.b)), nil
	case <-sigCh:
		_ = term.Restore(fd, state)
		fmt.Fprintln(os.Stderr)
		return "", ErrCancelled
	}
}
