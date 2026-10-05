package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
)

func TestDefaultEpicLine(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/OPS-40"):
			_, _ = io.WriteString(w, `{"key":"OPS-40","fields":{"issuetype":{"name":"Epic"},"summary":"DevOps k8s cluster upgrade"}}`)
		case strings.HasSuffix(r.URL.Path, "/OPS-41"):
			_, _ = io.WriteString(w, `{"key":"OPS-41","fields":{"issuetype":{"name":"Epic"},"summary":"Old","resolution":{"name":"Done"}}}`)
		case strings.HasSuffix(r.URL.Path, "/OPS-50"):
			time.Sleep(2 * time.Second)
			_, _ = io.WriteString(w, `{}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	client, err := jira.NewClientWith(&config.Config{Server: srv.URL}, "u", "t")
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		"":       "Default epic: none",
		"OPS-40": "Default epic: OPS-40 DevOps k8s cluster upgrade",
		"OPS-41": "Default epic: OPS-41 (done)",
		"OPS-12": "Default epic: OPS-12 (not found)",
		"OPS-50": "Default epic: OPS-50", // slow Jira: key only
	}
	for key, want := range cases {
		cfg := &config.Config{IssueDefaults: config.IssueDefaults{EpicLink: key}}
		start := time.Now()
		if got := defaultEpicLine(cfg, client); got != want {
			t.Errorf("%q: got %q, want %q", key, got, want)
		}
		if time.Since(start) > headerLookupTimeout+500*time.Millisecond {
			t.Errorf("%q: header waited too long", key)
		}
	}
}

func TestReviewMarksDefaultEpic(t *testing.T) {
	rows := reviewRows(&issueDraft{IssueType: "Task", Summary: "s", EpicLink: "OPS-40"}, nil, nil, "OPS-40")
	for _, r := range rows {
		if r.label == "Epic" && r.value != "OPS-40 (default)" {
			t.Fatalf("epic row %q", r.value)
		}
	}
	rows = reviewRows(&issueDraft{IssueType: "Task", Summary: "s"}, nil, nil, "OPS-40")
	for _, r := range rows {
		if r.label == "Epic" && r.value != "(none)" {
			t.Fatalf("epic row %q", r.value)
		}
	}
}
