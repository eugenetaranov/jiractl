package cmd

import (
	"os"
	"testing"

	"github.com/eugenetaranov/jiractl/internal/config"
)

func TestDraftRoundTrip(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if loadDraft() != nil {
		t.Fatal("expected no draft")
	}
	path, err := saveDraft(&issueDraft{Project: "P", IssueType: "Bug", Summary: "s", Description: "a\n\nb"})
	if err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", info.Mode().Perm())
	}
	d := loadDraft()
	if d == nil || d.Description != "a\n\nb" || d.SavedAt.IsZero() {
		t.Fatalf("got %+v", d)
	}
	deleteDraft()
	if loadDraft() != nil {
		t.Fatal("draft not deleted")
	}
}

func TestApplyDefaultsMergeOrder(t *testing.T) {
	cfg := &config.Config{IssueDefaults: config.IssueDefaults{
		IssueType:    "Task",
		EpicLink:     "OPS-1",
		Labels:       []string{"team"},
		CustomFields: map[string]string{"customfield_1": "a", "customfield_2": "b"},
	}}

	fresh := &issueDraft{Fields: map[string]string{"customfield_2": "draft"}}
	applyDefaults(fresh, cfg)
	if fresh.IssueType != "Task" || fresh.EpicLink != "OPS-1" || fresh.Fields["customfield_1"] != "a" || fresh.Fields["customfield_2"] != "draft" {
		t.Fatalf("fresh: %+v", fresh)
	}

	// A resumed draft keeps the user's choices, including "no epic".
	resumed := &issueDraft{IssueType: "Bug", DefaultsApplied: true, Fields: map[string]string{}}
	applyDefaults(resumed, cfg)
	if resumed.IssueType != "Bug" || resumed.EpicLink != "" || len(resumed.Fields) != 0 {
		t.Fatalf("resumed: %+v", resumed)
	}

	// Flags override both.
	if err := applyCreateFlags(fresh, createOptions{issueType: "Story", noEpic: true}, nil); err != nil {
		t.Fatal(err)
	}
	if fresh.IssueType != "Story" || fresh.EpicLink != "" {
		t.Fatalf("flags: %+v", fresh)
	}
}
