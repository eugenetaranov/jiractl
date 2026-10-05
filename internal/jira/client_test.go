package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eugenetaranov/jiractl/internal/config"
)

func newTestClient(t *testing.T, cfg *config.Config, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	cfg.Server = srv.URL
	c, err := NewClientWith(cfg, "u@example.com", "tok")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestResolveAccountID(t *testing.T) {
	c := newTestClient(t, &config.Config{}, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") != "john@example.com" {
			t.Errorf("query = %q", r.URL.Query().Get("query"))
		}
		io.WriteString(w, `[{"accountId":"abc123","displayName":"John","emailAddress":"john@example.com","active":true}]`)
	})
	if id, err := c.ResolveAccountID("john@example.com"); err != nil || id != "abc123" {
		t.Fatalf("got %q %v", id, err)
	}
}

func TestCreateIssueSendsComponentAndAccountID(t *testing.T) {
	var body map[string]interface{}
	c := newTestClient(t, &config.Config{}, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/2/issue":
			json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(201)
			io.WriteString(w, `{"key":"P-1"}`)
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
	})

	issue, err := c.CreateIssue(&NewIssue{Project: "P", Type: "Task", Summary: "s", Component: "Backend", AccountID: "abc123",
		Fields: map[string]string{"customfield_1": `{"value":"Ops"}`}})
	if err != nil {
		t.Fatal(err)
	}
	if issue.Key != "P-1" {
		t.Fatalf("key %q", issue.Key)
	}
	fields := body["fields"].(map[string]interface{})
	comps, _ := json.Marshal(fields["components"])
	if string(comps) != `[{"name":"Backend"}]` {
		t.Fatalf("components = %s", comps)
	}
	assignee, _ := json.Marshal(fields["assignee"])
	if string(assignee) != `{"accountId":"abc123"}` {
		t.Fatalf("assignee = %s", assignee)
	}
	cf, _ := json.Marshal(fields["customfield_1"])
	if string(cf) != `{"value":"Ops"}` {
		t.Fatalf("custom field = %s", cf)
	}
}

func TestResolveAccountIDAmbiguous(t *testing.T) {
	c := newTestClient(t, &config.Config{}, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `[{"accountId":"1","displayName":"Ann A","active":true},{"accountId":"2","displayName":"Ann B","active":true}]`)
	})
	_, err := c.ResolveAccountID("ann")
	if err == nil || !strings.Contains(err.Error(), "Ann A") || !strings.Contains(err.Error(), "Ann B") {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveAccountIDPassthrough(t *testing.T) {
	c := newTestClient(t, &config.Config{}, func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request expected")
	})
	id := "557058:f58131cb-b67d-43c7-b30d-6b58d40bd077"
	if got, err := c.ResolveAccountID(id); err != nil || got != id {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestAPIErrorFormatting(t *testing.T) {
	c := newTestClient(t, &config.Config{}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		io.WriteString(w, `{"errorMessages":[],"errors":{"summary":"Summary is required","parent":"Given parent is invalid"}}`)
	})
	_, err := c.CreateIssue(&NewIssue{Project: "P", Type: "Task"})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err %T %v", err, err)
	}
	if apiErr.Error() != "parent: Given parent is invalid; summary: Summary is required" {
		t.Fatalf("msg %q", apiErr.Error())
	}
	if !apiErr.HasField("parent") || strings.Contains(apiErr.Error(), "{") {
		t.Fatal("bad APIError")
	}
}

func TestSearchErrorIncludesJQLExplanation(t *testing.T) {
	c := newTestClient(t, &config.Config{}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		io.WriteString(w, `{"errorMessages":["Error in the JQL Query: Expecting operator but got 'x'."]}`)
	})
	_, err := c.SearchIssues("bad x", 10)
	if err == nil || !strings.Contains(err.Error(), "Expecting operator") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnauthorizedStatus(t *testing.T) {
	c := newTestClient(t, &config.Config{}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	})
	err := c.TestConnection()
	if StatusOf(err) != 401 || !strings.Contains(err.Error(), "authentication failed") {
		t.Fatalf("err = %v", err)
	}
}

func TestAnonymousClientServerInfo(t *testing.T) {
	c := newTestClientWith(t, "", "", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("anonymous client sent credentials")
		}
		io.WriteString(w, `{"version":"9.4.0","deploymentType":"Server"}`)
	})
	info, err := c.GetServerInfo()
	if err != nil || info.DeploymentType != "Server" {
		t.Fatalf("got %+v %v", info, err)
	}
}

func newTestClientWith(t *testing.T, user, token string, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := NewClientWith(&config.Config{Server: srv.URL}, user, token)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
