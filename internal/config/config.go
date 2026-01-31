package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	ConfigFileName = ".jiractl.toml"
)

type IssueDefaults struct {
	Assignee  string   `toml:"assignee,omitempty"`
	Component string   `toml:"component,omitempty"`
	EpicLink  string   `toml:"epic_link,omitempty"`
	IssueType string   `toml:"issue_type,omitempty"`
	Labels    []string `toml:"labels,omitempty"`
}

type Query struct {
	Name  string `toml:"name"`
	JQL   string `toml:"jql"`
	Limit int    `toml:"limit,omitempty"`
}

type Config struct {
	Server        string        `toml:"server"`
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

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
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
