package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// issueDraft holds what the user typed for a new issue, so nothing is lost
// when they decline the confirm step or Jira rejects the request.
type issueDraft struct {
	SavedAt     time.Time `json:"saved_at"`
	Project     string    `json:"project"`
	IssueType   string    `json:"issue_type"`
	Summary     string    `json:"summary"`
	Description string    `json:"description,omitempty"`
	EpicLink    string    `json:"epic_link,omitempty"`
	Assignee    string    `json:"assignee,omitempty"`
	Component   string    `json:"component,omitempty"`
	Labels      []string  `json:"labels,omitempty"`
	// DefaultsApplied is set once issue_defaults were copied into the draft,
	// so a resumed draft keeps the user's edits instead of re-applying them.
	DefaultsApplied bool `json:"defaults_applied,omitempty"`
	// Fields holds custom field values set for this issue (ID -> raw value),
	// on top of issue_defaults.custom_fields.
	Fields map[string]string `json:"fields,omitempty"`
}

func draftPath() (string, error) {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "jiractl", "draft.json"), nil
}

func saveDraft(d *issueDraft) (string, error) {
	path, err := draftPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	d.SavedAt = time.Now()
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// loadDraft returns the saved draft, or nil when there is none.
func loadDraft() *issueDraft {
	path, err := draftPath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var d issueDraft
	if json.Unmarshal(data, &d) != nil || d.Summary == "" {
		return nil
	}
	return &d
}

func deleteDraft() {
	if path, err := draftPath(); err == nil {
		_ = os.Remove(path)
	}
}

// keepDraft saves the draft and tells the user where it went.
func keepDraft(d *issueDraft) {
	path, err := saveDraft(d)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save draft: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "Draft saved to %s; run 'jiractl create' to resume it.\n", path)
}
