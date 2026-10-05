package cmd

import (
	"strings"
	"testing"

	"github.com/eugenetaranov/jiractl/internal/jira"
)

func TestFieldResolve(t *testing.T) {
	f := newFieldNames([]jira.Field{{ID: "customfield_10016", Name: "Story Points"}, {ID: "summary", Name: "Summary"}})
	for in, want := range map[string]string{"Story Points": "customfield_10016", "story points": "customfield_10016", "customfield_10016": "customfield_10016", "customfield_999": "customfield_999"} {
		if got, err := f.resolve(in); err != nil || got != want {
			t.Errorf("%q: got %q %v", in, got, err)
		}
	}
	_, err := f.resolve("Story Pints")
	if err == nil || !strings.Contains(err.Error(), "Story Points (customfield_10016)") {
		t.Fatalf("err = %v", err)
	}
	if f.label("customfield_10016") != "Story Points" || f.label("x") != "x" {
		t.Fatal("label")
	}
}

func TestParseFieldFlag(t *testing.T) {
	k, v, err := parseFieldFlag("Story Points=3=4")
	if err != nil || k != "Story Points" || v != "3=4" {
		t.Fatalf("%q %q %v", k, v, err)
	}
	if _, _, err := parseFieldFlag("novalue"); err == nil {
		t.Fatal("expected error")
	}
}
