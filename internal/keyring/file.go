package keyring

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

// fileCreds is the credentials file layout.
type fileCreds struct {
	Username string `toml:"username,omitempty"`
	Token    string `toml:"token,omitempty"`
}

func (c fileCreds) get(key string) string {
	if key == UsernameKey {
		return strings.TrimSpace(c.Username)
	}
	return strings.TrimSpace(c.Token)
}

// FilePath is the credentials file: $XDG_CONFIG_HOME/jiractl/credentials,
// falling back to ~/.config/jiractl/credentials.
func FilePath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "jiractl", "credentials"), nil
}

// DisplayPath is FilePath with the home directory shown as ~.
func DisplayPath() string {
	p, err := FilePath()
	if err != nil {
		return "credentials file"
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~" + p[len(home):]
	}
	return p
}

// FilePermError reports a credentials file readable by others, or nil.
func FilePermError() error {
	p, err := FilePath()
	if err != nil {
		return nil
	}
	info, err := os.Stat(p)
	if err != nil {
		return nil
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		return fmt.Errorf("%s has mode %04o, want 0600", DisplayPath(), mode)
	}
	return nil
}

var permWarnOnce sync.Once

func readFile() (fileCreds, error) {
	var c fileCreds
	p, err := FilePath()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, fmt.Errorf("failed to read %s: %w", DisplayPath(), err)
	}
	if err := FilePermError(); err != nil {
		permWarnOnce.Do(func() {
			fmt.Fprintf(os.Stderr, "Warning: %v (run: chmod 600 %s)\n", err, p)
		})
	}
	if _, err := toml.Decode(string(data), &c); err != nil {
		return c, fmt.Errorf("failed to parse %s: %w", DisplayPath(), err)
	}
	return c, nil
}

// setFile writes one value, keeping the other, through a temp file and
// rename so a crash never leaves a half-written file.
func setFile(key, value string) error {
	c, err := readFile()
	if err != nil {
		return err
	}
	if key == UsernameKey {
		c.Username = value
	} else {
		c.Token = value
	}
	p, err := FilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return fmt.Errorf("failed to create %s: %w", filepath.Dir(p), err)
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(c); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".credentials-*")
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", DisplayPath(), err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write %s: %w", DisplayPath(), err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		return fmt.Errorf("failed to write %s: %w", DisplayPath(), err)
	}
	return nil
}

func removeFile() error {
	p, err := FilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to delete %s: %w", DisplayPath(), err)
	}
	return nil
}
