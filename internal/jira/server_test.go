package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/eugenetaranov/jiractl/internal/config"
)

func newServerClient(t *testing.T, cfg *config.Config, h http.HandlerFunc) *Client {
	t.Helper()
	cfg.Deployment = config.DeploymentServer
	return newTestClient(t, cfg, h)
}

func TestServerUsesBearerAndV2Search(t *testing.T) {
	c := newServerClient(t, &config.Config{}, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q", got)
		}
		if r.URL.Path != "/rest/api/2/search" {
			t.Errorf("path = %s", r.URL.Path)
		}
		io.WriteString(w, `{"issues":[{"key":"OPS-1","fields":{"summary":"s","description":"plain text"}}]}`)
	})
	issues, err := c.SearchIssues("project = OPS", 10)
	if err != nil || len(issues) != 1 || issues[0].Fields.Description != "plain text" {
		t.Fatalf("got %+v %v", issues, err)
	}
}

func TestNewClientServerNeedsNoUsername(t *testing.T) {
	// NewClient reads the keyring; NewClientWith is what it delegates to.
	c, err := NewClientWith(&config.Config{Server: "http://x", Deployment: config.DeploymentServer}, "", "tok")
	if err != nil || c == nil {
		t.Fatal(err)
	}
}

func createMetaHandler(t *testing.T, fields string, body *map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/2/issue/createmeta/OPS/issuetypes":
			io.WriteString(w, `{"values":[{"id":"10","name":"Task"}]}`)
		case "/rest/api/2/issue/createmeta/OPS/issuetypes/10":
			io.WriteString(w, `{"values":`+fields+`}`)
		case "/rest/api/2/issue":
			json.NewDecoder(r.Body).Decode(body)
			w.WriteHeader(201)
			io.WriteString(w, `{"key":"OPS-2"}`)
		default:
			t.Errorf("unexpected %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}
}

func TestEpicLinkFieldFallback(t *testing.T) {
	var body map[string]interface{}
	c := newServerClient(t, &config.Config{}, createMetaHandler(t,
		`[{"fieldId":"summary","name":"Summary"},{"fieldId":"customfield_10014","name":"Epic Link","schema":{"custom":"com.pyxis.greenhopper.jira:gh-epic-link"}}]`, &body))

	field := c.EpicField("OPS", "Task")
	if field != "customfield_10014" {
		t.Fatalf("field = %q", field)
	}
	if _, err := c.CreateIssue(&NewIssue{Project: "OPS", Type: "Task", Summary: "s", EpicLink: "OPS-40", EpicField: field, AssigneeName: "jdoe"}); err != nil {
		t.Fatal(err)
	}
	fields := body["fields"].(map[string]interface{})
	if fields["customfield_10014"] != "OPS-40" || fields["parent"] != nil {
		t.Fatalf("fields = %v", fields)
	}
	assignee, _ := json.Marshal(fields["assignee"])
	if string(assignee) != `{"name":"jdoe"}` {
		t.Fatalf("assignee = %s", assignee)
	}
}

func TestEpicFieldParentAndNone(t *testing.T) {
	var body map[string]interface{}
	c := newServerClient(t, &config.Config{}, createMetaHandler(t, `[{"fieldId":"parent","name":"Parent"}]`, &body))
	if f := c.EpicField("OPS", "Task"); f != "parent" {
		t.Fatalf("field = %q", f)
	}

	c = newServerClient(t, &config.Config{}, createMetaHandler(t, `[{"fieldId":"summary","name":"Summary"}]`, &body))
	if f := c.EpicField("OPS", "Task"); f != "" {
		t.Fatalf("field = %q, want none", f)
	}

	c = newServerClient(t, &config.Config{IssueDefaults: config.IssueDefaults{EpicField: "customfield_1"}}, func(w http.ResponseWriter, r *http.Request) {
		t.Error("override must not call Jira")
	})
	if f := c.EpicField("OPS", "Task"); f != "customfield_1" {
		t.Fatalf("override = %q", f)
	}
}

func TestParseJQLFallsBackOnServer(t *testing.T) {
	c := newServerClient(t, &config.Config{}, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/rest/api/3/jql/parse") {
			w.WriteHeader(404)
			return
		}
		if strings.Contains(r.URL.Query().Get("jql"), "bad") {
			w.WriteHeader(400)
			io.WriteString(w, `{"errorMessages":["Field 'bad' does not exist."]}`)
			return
		}
		io.WriteString(w, `{"issues":[]}`)
	})
	res, err := c.ParseJQL([]string{"project = OPS", "bad = 1"})
	if err != nil || len(res[0]) != 0 || len(res[1]) != 1 || !strings.Contains(res[1][0], "does not exist") {
		t.Fatalf("got %v %v", res, err)
	}
}
