package jira

import (
	"encoding/json"
	"testing"

	jira "github.com/andygrunwald/go-jira"
)

func issueJSON(t *testing.T, s string) jira.Issue {
	t.Helper()
	var is jira.Issue
	if err := json.Unmarshal([]byte(s), &is); err != nil {
		t.Fatal(err)
	}
	return is
}

func TestSortForDisplay(t *testing.T) {
	issues := []jira.Issue{
		issueJSON(t, `{"key":"DONE-NEW","fields":{"status":{"name":"Completed","statusCategory":{"key":"done"}},"updated":"2026-10-05T10:00:00.000+0000"}}`),
		issueJSON(t, `{"key":"TODO-OLD","fields":{"status":{"name":"To Do","statusCategory":{"key":"new"}},"updated":"2026-09-01T10:00:00.000+0000"}}`),
		issueJSON(t, `{"key":"PROG","fields":{"status":{"name":"In Review","statusCategory":{"key":"indeterminate"}},"updated":"2026-08-01T10:00:00.000+0000"}}`),
		issueJSON(t, `{"key":"TODO-NEW","fields":{"status":{"name":"Backlog","statusCategory":{"key":"new"}},"updated":"2026-10-04T10:00:00.000+0000"}}`),
		issueJSON(t, `{"key":"RESOLVED","fields":{"resolution":{"name":"Done"},"updated":"2026-10-05T11:00:00.000+0000"}}`),
	}
	SortForDisplay(issues)
	var got []string
	for _, is := range issues {
		got = append(got, is.Key)
	}
	want := []string{"PROG", "TODO-NEW", "TODO-OLD", "RESOLVED", "DONE-NEW"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
