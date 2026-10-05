package doctor

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eugenetaranov/jiractl/internal/config"
)

// fakeJira serves a healthy instance; tweak flips individual answers.
type fakeJira struct {
	authFails   bool
	noTransit   bool
	noCreate    bool
	badJQL      bool
	components  []string
	createField string
}

func (f *fakeJira) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	switch {
	case p == "/rest/api/2/serverInfo":
		io.WriteString(w, `{"version":"1001.0.0","deploymentType":"Cloud"}`)
	case f.authFails:
		w.WriteHeader(401)
	case p == "/rest/api/2/myself":
		io.WriteString(w, `{"accountId":"me","displayName":"Eugene","emailAddress":"e@example.com"}`)
	case p == "/rest/api/2/project/OPS":
		io.WriteString(w, `{"key":"OPS","issueTypes":[{"id":"1","name":"Task"},{"id":"2","name":"Bug"}]}`)
	case p == "/rest/api/2/mypermissions":
		perm := func(b bool) string {
			if b {
				return `{"havePermission":true}`
			}
			return `{"havePermission":false}`
		}
		io.WriteString(w, `{"permissions":{"BROWSE_PROJECTS":`+perm(true)+`,"CREATE_ISSUES":`+perm(!f.noCreate)+
			`,"ASSIGN_ISSUES":`+perm(true)+`,"TRANSITION_ISSUES":`+perm(!f.noTransit)+`,"ADD_COMMENTS":`+perm(true)+`}}`)
	case p == "/rest/api/2/issue/OPS-40":
		io.WriteString(w, `{"key":"OPS-40","fields":{"issuetype":{"name":"Epic"},"summary":"k8s"}}`)
	case p == "/rest/api/2/project/OPS/components":
		b, _ := json.Marshal(func() []map[string]string {
			var out []map[string]string
			for _, c := range f.components {
				out = append(out, map[string]string{"name": c})
			}
			return out
		}())
		w.Write(b)
	case p == "/rest/api/2/issue/createmeta/OPS/issuetypes":
		io.WriteString(w, `{"issueTypes":[{"id":"1","name":"Task"}]}`)
	case p == "/rest/api/2/issue/createmeta/OPS/issuetypes/1":
		field := f.createField
		if field == "" {
			field = "customfield_15838"
		}
		io.WriteString(w, `{"fields":[{"fieldId":"summary","name":"Summary","required":true},
			{"fieldId":"`+field+`","name":"Work Allocation","required":true,"allowedValues":[{"value":"Operations"},{"value":"Projects"}]}]}`)
	case p == "/rest/api/3/jql/parse":
		if f.badJQL {
			io.WriteString(w, `{"queries":[{"errors":[]},{"errors":["Error in the JQL Query: Expecting a date but got '-7x'."]}]}`)
		} else {
			io.WriteString(w, `{"queries":[{"errors":[]},{"errors":[]}]}`)
		}
	default:
		w.WriteHeader(404)
		io.WriteString(w, `{"errorMessages":["no stub for `+p+`"]}`)
	}
}

func runFake(t *testing.T, f *fakeJira, mutate func(*config.Config)) map[string]Result {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	cfg := &config.Config{
		Server:  srv.URL,
		Project: "OPS",
		IssueDefaults: config.IssueDefaults{
			IssueType:    "Task",
			EpicLink:     "OPS-40",
			CustomFields: map[string]string{"customfield_15838": `{"value": "Operations"}`},
		},
		Queries: []config.Query{{Name: "mine", JQL: "assignee = currentUser()"}, {Name: "recent", JQL: "updated >= -7d"}},
	}
	if mutate != nil {
		mutate(cfg)
	}
	ctx := &Ctx{Cfg: cfg, Username: "e@example.com", Token: "tok"}
	byID := map[string]Result{}
	for _, r := range Run(ctx, Checks(false)) {
		byID[r.ID] = r
	}
	return byID
}

func TestHealthy(t *testing.T) {
	res := runFake(t, &fakeJira{}, nil)
	for id, r := range res {
		if r.Status != OK {
			t.Errorf("%s: %v %s", id, r.Status, r.Detail)
		}
	}
	var list []Result
	for _, r := range res {
		list = append(list, r)
	}
	if Failed(list) {
		t.Fatal("Failed() on healthy setup")
	}
}

func TestAuthFailureSkipsDependents(t *testing.T) {
	res := runFake(t, &fakeJira{authFails: true}, nil)
	if res["auth"].Status != Fail || !strings.Contains(res["auth"].Hint, "jiractl configure") {
		t.Fatalf("auth: %+v", res["auth"])
	}
	for _, id := range []string{"project", "perm.create", "default.epic", "default.fields", "queries"} {
		if res[id].Status != Skip {
			t.Errorf("%s: %v, want skip", id, res[id].Status)
		}
	}
	if res["default.fields"].Detail != "requires Default issue type" {
		t.Errorf("skip detail %q", res["default.fields"].Detail)
	}
}

func TestPermissionSeverity(t *testing.T) {
	res := runFake(t, &fakeJira{noTransit: true}, nil)
	if res["perm.transition"].Status != Warn {
		t.Errorf("transition: %v", res["perm.transition"].Status)
	}
	res = runFake(t, &fakeJira{noCreate: true}, nil)
	if res["perm.create"].Status != Fail || !strings.Contains(res["perm.create"].Hint, "OPS") {
		t.Errorf("create: %+v", res["perm.create"])
	}
}

func TestCustomFieldNotOnScreen(t *testing.T) {
	res := runFake(t, &fakeJira{createField: "customfield_99"}, nil)
	r := res["default.fields"]
	if r.Status != Fail || !strings.Contains(r.Detail, "customfield_15838 is not on the create screen") {
		t.Fatalf("%+v", r)
	}
}

func TestCustomFieldBadOption(t *testing.T) {
	res := runFake(t, &fakeJira{}, func(c *config.Config) {
		c.IssueDefaults.CustomFields["customfield_15838"] = `{"value": "Ops"}`
	})
	r := res["default.fields"]
	if r.Status != Fail || !strings.Contains(r.Detail, `Work Allocation (customfield_15838): "Ops" is not an allowed value`) {
		t.Fatalf("%+v", r)
	}
}

func TestRequiredFieldWithoutDefault(t *testing.T) {
	res := runFake(t, &fakeJira{}, func(c *config.Config) { c.IssueDefaults.CustomFields = nil })
	r := res["default.fields"]
	if r.Status != Warn || !strings.Contains(r.Detail, "Work Allocation (customfield_15838)") {
		t.Fatalf("%+v", r)
	}
}

func TestBrokenJQL(t *testing.T) {
	res := runFake(t, &fakeJira{badJQL: true}, nil)
	r := res["queries"]
	if r.Status != Fail || !strings.Contains(r.Detail, "recent: Error in the JQL Query: Expecting a date but got '-7x'.") {
		t.Fatalf("%+v", r)
	}
}

func TestComponentSuggestion(t *testing.T) {
	res := runFake(t, &fakeJira{components: []string{"Backend", "Frontend"}}, func(c *config.Config) {
		c.IssueDefaults.Component = "Backnd"
	})
	r := res["default.component"]
	if r.Status != Fail || r.Hint != `set issue_defaults.component = "Backend"` {
		t.Fatalf("%+v", r)
	}
}

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	PrintJSON(&buf, []Result{{ID: "auth", Title: "Authentication", Status: Fail, Detail: "x", Hint: "y"}})
	var out []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil || out[0]["status"] != "fail" || out[0]["id"] != "auth" {
		t.Fatalf("%s %v", buf.String(), err)
	}
}
