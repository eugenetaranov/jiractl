package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

const (
	ConfigFileName = ".jiractl.toml"
)

type IssueDefaults struct {
	Assignee  string `toml:"assignee,omitempty"`
	Component string `toml:"component,omitempty"`
	EpicLink  string `toml:"epic_link,omitempty"`
	// EpicField overrides how the epic is linked: "parent" or the ID of the
	// Epic Link custom field. Detected from the create screen when empty.
	EpicField    string            `toml:"epic_field,omitempty"`
	IssueType    string            `toml:"issue_type,omitempty"`
	Labels       []string          `toml:"labels,omitempty"`
	CustomFields map[string]string `toml:"custom_fields,omitempty"`
}

type Query struct {
	Name  string `toml:"name"`
	JQL   string `toml:"jql"`
	Limit int    `toml:"limit,omitempty"`
}

// Deployment values; an empty Deployment means DeploymentCloud.
const (
	DeploymentCloud  = "cloud"
	DeploymentServer = "server" // Server or Data Center
)

type Config struct {
	Server        string        `toml:"server"`
	Deployment    string        `toml:"deployment,omitempty"`
	Project       string        `toml:"project"`
	IssueDefaults IssueDefaults `toml:"issue_defaults,omitempty"`
	Queries       []Query       `toml:"queries,omitempty"`
}

func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ConfigFileName), nil
}

// warnedKeys makes sure each unknown key is reported once per process even
// though the config is loaded by several code paths.
var (
	warnedMu   sync.Mutex
	warnedKeys = map[string]bool{}
)

func Load() (*Config, error) {
	return load(true)
}

// LoadQuiet loads the config without warning about unknown keys, for callers
// that report them themselves.
func LoadQuiet() (*Config, error) {
	return load(false)
}

func load(warnUnknown bool) (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	md, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if !warnUnknown {
		return cfg, nil
	}
	warnedMu.Lock()
	for _, key := range md.Undecoded() {
		k := key.String()
		if !warnedKeys[k] {
			warnedKeys[k] = true
			fmt.Fprintf(os.Stderr, "warning: unknown config key %q in ~/%s\n", k, ConfigFileName)
		}
	}
	warnedMu.Unlock()

	return cfg, nil
}

// Save writes the config to disk. Only keys whose values changed are touched,
// so comments and formatting in the existing file survive. The write is
// atomic (temp file + rename) and the file is private (0600).
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return c.saveTo(path)
}

func (c *Config) saveTo(path string) error {
	original, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var out []byte
	if len(original) == 0 {
		out, err = encode(c)
		if err != nil {
			return err
		}
	} else {
		patched, ok := patchConfig(string(original), c)
		if ok {
			out = []byte(patched)
		} else {
			// The file has a layout the patcher can't edit safely (inline
			// tables, multi-line arrays, changed queries...). Keep a backup
			// and rewrite it from scratch.
			backup := path + ".bak"
			if err := writeAtomic(backup, original); err != nil {
				return fmt.Errorf("failed to write config backup: %w", err)
			}
			fmt.Fprintf(os.Stderr, "warning: rewrote ~/%s from scratch; previous version saved to %s\n", ConfigFileName, backup)
			out, err = encode(c)
			if err != nil {
				return err
			}
		}
	}

	if err := writeAtomic(path, out); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

func encode(c *Config) ([]byte, error) {
	var sb strings.Builder
	if err := toml.NewEncoder(&sb).Encode(c); err != nil {
		return nil, fmt.Errorf("failed to encode config: %w", err)
	}
	return []byte(sb.String()), nil
}

// writeAtomic writes data to a temp file in the same directory and renames it
// over path, so a failed write never leaves a truncated file behind.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}

	if err := tmp.Chmod(0o600); err != nil {
		cleanup()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

// IsServer reports whether the instance is Jira Server or Data Center.
func (c *Config) IsServer() bool {
	return c.Deployment == DeploymentServer
}

// ExpandJQL replaces ${project} placeholder with the actual project key
func (c *Config) ExpandJQL(jql string) string {
	return strings.ReplaceAll(jql, "${project}", c.Project)
}

// GetQuery returns a query by name
func (c *Config) GetQuery(name string) *Query {
	for i := range c.Queries {
		if c.Queries[i].Name == name {
			return &c.Queries[i]
		}
	}
	return nil
}

// QueryNames returns a list of all query names
func (c *Config) QueryNames() []string {
	names := make([]string, len(c.Queries))
	for i, q := range c.Queries {
		names[i] = q.Name
	}
	return names
}

// UnknownKeys returns the keys in the config file that jiractl doesn't use,
// without printing warnings.
func UnknownKeys() ([]string, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	var cfg Config
	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, k := range md.Undecoded() {
		keys = append(keys, k.String())
	}
	return keys, nil
}
