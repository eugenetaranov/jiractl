package jira

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/eugenetaranov/jiractl/internal/config"
)

func TestEpicSearchJQL(t *testing.T) {
	cases := []struct{ text, want string }{
		{"", `project = OPS AND issuetype = Epic AND resolution = Unresolved ORDER BY updated DESC`},
		{"devops k8s", `project = OPS AND issuetype = Epic AND (summary ~ "devops*" AND summary ~ "k8s*") ORDER BY updated DESC`},
		{"ops-40", `project = OPS AND issuetype = Epic AND ((summary ~ "ops-40*") OR key = "OPS-40") ORDER BY updated DESC`},
		{`say "hi"`, `project = OPS AND issuetype = Epic AND (summary ~ "say*" AND summary ~ "hi*") ORDER BY updated DESC`},
	}
	for _, c := range cases {
		if got := EpicSearchJQL("OPS", c.text, true); got != c.want {
			t.Errorf("%q:\n got %s\nwant %s", c.text, got, c.want)
		}
	}
}

func TestSearchEpicsFallsBackAndSortsOpenFirst(t *testing.T) {
	calls := 0
	c := newTestClient(t, &config.Config{}, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if strings.Contains(r.URL.Query().Get("jql"), "*") {
			w.WriteHeader(400)
			io.WriteString(w, `{"errorMessages":["wildcards not allowed"]}`)
			return
		}
		io.WriteString(w, `{"issues":[
			{"key":"OPS-1","fields":{"summary":"done one","resolution":{"name":"Done"}}},
			{"key":"OPS-2","fields":{"summary":"open one"}}]}`)
	})
	epics, err := c.SearchEpics("OPS", "one")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || epics[0].Key != "OPS-2" || epics[1].Key != "OPS-1" {
		t.Fatalf("calls=%d order=%s,%s", calls, epics[0].Key, epics[1].Key)
	}
}

func TestCheckEpic(t *testing.T) {
	c := newTestClient(t, &config.Config{}, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/OPS-404"):
			w.WriteHeader(404)
			io.WriteString(w, `{"errorMessages":["Issue does not exist"]}`)
		case strings.HasSuffix(r.URL.Path, "/OPS-5"):
			io.WriteString(w, `{"key":"OPS-5","fields":{"issuetype":{"name":"Story"}}}`)
		default:
			io.WriteString(w, `{"key":"OPS-40","fields":{"issuetype":{"name":"Epic"},"summary":"k8s"}}`)
		}
	})
	for key, want := range map[string]EpicState{"OPS-404": EpicNotFound, "OPS-5": EpicNotAnEpic, "OPS-40": EpicOK} {
		got, _, err := c.CheckEpic(key)
		if err != nil || got != want {
			t.Errorf("%s: got %v %v, want %v", key, got, err, want)
		}
	}
}

func TestIsEpicRejection(t *testing.T) {
	if !IsEpicRejection(&APIError{Status: 400, Fields: map[string]string{"parent": "Given parent is invalid"}}) {
		t.Error("parent not detected")
	}
	if !IsEpicRejection(&APIError{Status: 400, Fields: map[string]string{"customfield_10014": "Epic Link is invalid"}}) {
		t.Error("epic link not detected")
	}
	if IsEpicRejection(&APIError{Status: 400, Fields: map[string]string{"summary": "required"}}) {
		t.Error("false positive")
	}
}
