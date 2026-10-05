package cmd

import (
	"os"
	"testing"
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
