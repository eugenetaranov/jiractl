package cmd

import (
	"strings"
	"testing"
)

func TestCreatedIssuePrint(t *testing.T) {
	c := &createdIssue{
		key: "ABC-1",
		url: "https://jira.example.com/browse/ABC-1",
		rows: []reviewRow{
			{"Type", "Bug"},
			{"Summary", strings.Repeat("long summary ", 20)},
			{"Description", "line one\nline two"},
		},
	}
	got := c.report()
	want := "Created issue ABC-1\n\n" +
		"  Type:         Bug\n" +
		"  Summary:      " + strings.Repeat("long summary ", 20) + "\n" +
		"  Description:  line one\n" +
		"                line two\n" +
		"\nhttps://jira.example.com/browse/ABC-1\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
