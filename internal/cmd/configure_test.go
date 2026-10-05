package cmd

import "testing"

func TestNormalizeServer(t *testing.T) {
	for in, want := range map[string]string{
		"acme.atlassian.net":            "https://acme.atlassian.net",
		" https://acme.atlassian.net/ ": "https://acme.atlassian.net",
		"http://jira.local:8080":        "http://jira.local:8080",
		"":                              "",
	} {
		if got := normalizeServer(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}
