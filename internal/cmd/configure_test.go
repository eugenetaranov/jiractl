package cmd

import (
	"testing"
	"time"
)

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

func TestShortAge(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	for d, want := range map[time.Duration]string{
		30 * time.Second:     "now",
		5 * time.Minute:      "5m",
		3 * time.Hour:        "3h",
		2 * 24 * time.Hour:   "2d",
		21 * 24 * time.Hour:  "3w",
		120 * 24 * time.Hour: "4mo",
		400 * 24 * time.Hour: "1y",
	} {
		if got := shortAge(now.Add(-d), now); got != want {
			t.Errorf("%v: got %q, want %q", d, got, want)
		}
	}
	if shortAge(time.Time{}, now) != "-" {
		t.Error("zero time")
	}
}
