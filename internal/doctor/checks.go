package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	jiralib "github.com/andygrunwald/go-jira"
	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/keyring"
)

const configureHint = "run 'jiractl configure'"

// Ctx carries what the checks share: config, credentials, the client and
// cached lookups.
type Ctx struct {
	Cfg        *config.Config
	CfgErr     error
	ConfigPath string
	Username   string
	Token      string
	Client     *jira.Client

	issueTypes []jiralib.IssueType

	permsOnce sync.Once
	perms     map[string]bool
	permsErr  error
}

// NewCtx loads the config and credentials. Problems are recorded for the
// checks to report rather than returned.
func NewCtx() *Ctx {
	ctx := &Ctx{}
	ctx.ConfigPath, _ = config.ConfigPath()
	ctx.Cfg, ctx.CfgErr = config.LoadQuiet()
	ctx.Username, ctx.Token, _ = keyring.GetCredentials()
	return ctx
}

var permissionNames = []string{"BROWSE_PROJECTS", "CREATE_ISSUES", "ASSIGN_ISSUES", "TRANSITION_ISSUES", "ADD_COMMENTS"}

func (ctx *Ctx) permissions() (map[string]bool, error) {
	ctx.permsOnce.Do(func() {
		ctx.perms, ctx.permsErr = ctx.Client.MyPermissions(ctx.Cfg.Project, permissionNames)
	})
	return ctx.perms, ctx.permsErr
}

// Checks returns every check in order. withConfig=false leaves out the
// config file checks (used after configure, which just wrote it).
func Checks(withConfig bool) []Check {
	var checks []Check
	if withConfig {
		checks = append(checks,
			Check{ID: "config", Title: "Config file", Run: checkConfig},
			Check{ID: "config.perms", Title: "Config permissions", DependsOn: []string{"config"}, Run: checkConfigPerms},
			Check{ID: "config.keys", Title: "Config keys", DependsOn: []string{"config"}, Run: checkConfigKeys},
		)
	}
	checks = append(checks,
		Check{ID: "creds", Title: "Credentials", Run: checkCreds},
		Check{ID: "server", Title: "Server", DependsOn: []string{"config"}, Run: checkServer},
		Check{ID: "auth", Title: "Authentication", DependsOn: []string{"creds", "server"}, Run: checkAuth},
		Check{ID: "project", Title: "Project", DependsOn: []string{"auth"}, Run: checkProject},
		Check{ID: "default.type", Title: "Default issue type", DependsOn: []string{"project"}, Run: checkDefaultType},
		permCheck("perm.browse", "Browse project", "BROWSE_PROJECTS", Fail, "queries and create need it"),
		permCheck("perm.create", "Create issues", "CREATE_ISSUES", Fail, "jiractl create needs it"),
		permCheck("perm.assign", "Assign issues", "ASSIGN_ISSUES", Warn, "'Assign to me' won't work"),
		permCheck("perm.transition", "Transition issues", "TRANSITION_ISSUES", Warn, "'Transition' won't work"),
		permCheck("perm.comment", "Add comments", "ADD_COMMENTS", Warn, "'Comment' won't work"),
		Check{ID: "default.epic", Title: "Default epic", DependsOn: []string{"project"}, Parallel: true, Run: checkDefaultEpic},
		Check{ID: "default.component", Title: "Default component", DependsOn: []string{"project"}, Parallel: true, Run: checkDefaultComponent},
		Check{ID: "default.assignee", Title: "Default assignee", DependsOn: []string{"auth"}, Parallel: true, Run: checkDefaultAssignee},
		Check{ID: "default.fields", Title: "Custom fields", DependsOn: []string{"default.type"}, Parallel: true, Run: checkDefaultFields},
		Check{ID: "queries", Title: "Saved queries", DependsOn: []string{"project"}, Parallel: true, Run: checkQueries},
	)
	return checks
}

func checkConfig(ctx *Ctx) Result {
	if _, err := os.Stat(ctx.ConfigPath); os.IsNotExist(err) {
		return fail("~/"+config.ConfigFileName+" not found", configureHint)
	}
	if ctx.CfgErr != nil {
		return fail(ctx.CfgErr.Error(), "fix the TOML syntax, or "+configureHint)
	}
	if ctx.Cfg.Server == "" || ctx.Cfg.Project == "" {
		return fail("server or project is not set", configureHint)
	}
	return ok("~/" + config.ConfigFileName)
}

func checkConfigPerms(ctx *Ctx) Result {
	info, err := os.Stat(ctx.ConfigPath)
	if err != nil {
		return warn(err.Error(), "")
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		return warn(fmt.Sprintf("mode %04o, readable by others", mode), "chmod 600 ~/"+config.ConfigFileName)
	}
	return ok("0600")
}

func checkConfigKeys(ctx *Ctx) Result {
	keys, err := config.UnknownKeys()
	if err != nil {
		return warn(err.Error(), "")
	}
	if len(keys) > 0 {
		return warn("unknown keys: "+strings.Join(keys, ", "), "fix or remove them; jiractl ignores them")
	}
	return ok("")
}

func checkCreds(ctx *Ctx) Result {
	switch {
	case ctx.Username == "" && ctx.Token == "":
		return fail("no username or API token in the system keyring", configureHint)
	case ctx.Token == "":
		return fail("no API token in the system keyring", configureHint)
	case ctx.Username == "" && (ctx.Cfg == nil || !ctx.Cfg.IsServer()):
		return fail("no username in the system keyring", configureHint)
	}
	if ctx.Username == "" {
		return ok("personal access token")
	}
	return ok(ctx.Username)
}

func checkServer(ctx *Ctx) Result {
	if ctx.Cfg == nil || ctx.Cfg.Server == "" {
		return fail("server is not set", configureHint)
	}
	anon, err := jira.NewClientWith(ctx.Cfg, "", "")
	if err != nil {
		return fail(err.Error(), "check 'server' in ~/"+config.ConfigFileName)
	}
	info, err := anon.GetServerInfo()
	if err != nil {
		return fail(fmt.Sprintf("%s: %v", ctx.Cfg.Server, err), "check 'server' in ~/"+config.ConfigFileName+" and your network")
	}
	detail := strings.TrimSpace(fmt.Sprintf("%s %s %s", ctx.Cfg.Server, info.DeploymentType, info.Version))
	isServer := info.DeploymentType != "" && !strings.EqualFold(info.DeploymentType, "Cloud")
	if isServer != ctx.Cfg.IsServer() {
		configured := config.DeploymentCloud
		if ctx.Cfg.IsServer() {
			configured = config.DeploymentServer
		}
		return warn(fmt.Sprintf("%s, but the config says deployment = %q", detail, configured),
			"run 'jiractl configure' to switch the authentication and APIs used")
	}
	return ok(detail)
}

func checkAuth(ctx *Ctx) Result {
	client, err := jira.NewClientWith(ctx.Cfg, ctx.Username, ctx.Token)
	if err != nil {
		return fail(err.Error(), configureHint)
	}
	me, err := client.Myself()
	if err != nil {
		if s := jira.StatusOf(err); s == 401 || s == 403 {
			return fail("credentials rejected", "create a new API token and "+configureHint)
		}
		return fail(err.Error(), "")
	}
	ctx.Client = client
	who := me.DisplayName
	if me.EmailAddress != "" {
		who += " <" + me.EmailAddress + ">"
	}
	return ok(who)
}

func checkProject(ctx *Ctx) Result {
	types, err := ctx.Client.GetIssueTypes(ctx.Cfg.Project)
	if err != nil {
		if jira.StatusOf(err) == 404 {
			return fail(fmt.Sprintf("project %s not found or not visible to you", ctx.Cfg.Project), "pick another project with 'jiractl configure'")
		}
		return fail(err.Error(), "")
	}
	ctx.issueTypes = types
	return ok(fmt.Sprintf("%s (%d issue types)", ctx.Cfg.Project, len(types)))
}

func permCheck(id, title, perm string, severity Status, consequence string) Check {
	return Check{ID: id, Title: title, DependsOn: []string{"project"}, Parallel: true, Run: func(ctx *Ctx) Result {
		perms, err := ctx.permissions()
		if err != nil {
			return warn("could not read permissions: "+err.Error(), "")
		}
		if perms[perm] {
			return ok("")
		}
		detail := fmt.Sprintf("missing %s in %s; %s", perm, ctx.Cfg.Project, consequence)
		return Result{Status: severity, Detail: detail, Hint: fmt.Sprintf("ask a Jira admin for the %q permission in %s", title, ctx.Cfg.Project)}
	}}
}

func checkDefaultType(ctx *Ctx) Result {
	want := ctx.Cfg.IssueDefaults.IssueType
	if want == "" {
		return ok("not set; asked on create")
	}
	var names []string
	for _, t := range ctx.issueTypes {
		if strings.EqualFold(t.Name, want) {
			return ok(t.Name)
		}
		names = append(names, t.Name)
	}
	return fail(fmt.Sprintf("%q is not an issue type in %s", want, ctx.Cfg.Project),
		"set issue_defaults.issue_type to one of: "+strings.Join(names, ", "))
}

func checkDefaultEpic(ctx *Ctx) Result {
	key := ctx.Cfg.IssueDefaults.EpicLink
	if key == "" {
		return ok("not set")
	}
	const hint = "choose another with 'Change default epic' in the jiractl menu, or set issue_defaults.epic_link"
	state, issue, err := ctx.Client.CheckEpic(key)
	switch {
	case err != nil:
		return warn(err.Error(), "")
	case state == jira.EpicNotFound:
		return warn(key+" not found", hint)
	case state == jira.EpicNotAnEpic:
		return warn(key+" is not an epic", hint)
	case jira.IsResolved(*issue):
		return warn(key+" is done", hint)
	}
	return ok(key + " " + issue.Fields.Summary)
}

func checkDefaultComponent(ctx *Ctx) Result {
	want := ctx.Cfg.IssueDefaults.Component
	if want == "" {
		return ok("not set")
	}
	names, err := ctx.Client.ListComponents(ctx.Cfg.Project)
	if err != nil {
		return warn("could not list components: "+err.Error(), "")
	}
	for _, n := range names {
		if n == want {
			return ok(n)
		}
	}
	hint := "set issue_defaults.component to an existing component"
	if s := closest(want, names); s != "" {
		hint = fmt.Sprintf("set issue_defaults.component = %q", s)
	} else if len(names) > 0 {
		hint += ": " + strings.Join(names, ", ")
	}
	return fail(fmt.Sprintf("component %q doesn't exist in %s", want, ctx.Cfg.Project), hint)
}

func checkDefaultAssignee(ctx *Ctx) Result {
	want := ctx.Cfg.IssueDefaults.Assignee
	if want == "" {
		return ok("not set")
	}
	if ctx.Cfg.IsServer() {
		return ok(want + " (username)")
	}
	id, err := ctx.Client.ResolveAccountID(want)
	if err != nil {
		return fail(err.Error(), "set issue_defaults.assignee to an email, name or account ID")
	}
	return ok(fmt.Sprintf("%s (%s)", want, id))
}

// standardFields are always provided by jiractl.
var standardFields = map[string]bool{"summary": true, "issuetype": true, "project": true, "reporter": true, "description": true, "parent": true}

func checkDefaultFields(ctx *Ctx) Result {
	d := ctx.Cfg.IssueDefaults
	if d.IssueType == "" {
		if len(d.CustomFields) > 0 {
			return warn("can't check custom fields without a default issue type", "set issue_defaults.issue_type")
		}
		return ok("nothing to check")
	}
	fields, err := ctx.Client.CreateMetaFields(ctx.Cfg.Project, d.IssueType)
	if err != nil {
		return warn("could not read the create screen: "+err.Error(), "")
	}
	byID := map[string]jira.CreateField{}
	for _, f := range fields {
		byID[f.ID] = f
	}

	var problems []string
	ids := make([]string, 0, len(d.CustomFields))
	for id := range d.CustomFields {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		f, ok := byID[id]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s is not on the create screen for %s", id, d.IssueType))
			continue
		}
		if v := configuredOption(d.CustomFields[id]); v != "" && len(f.AllowedValues) > 0 && !contains(f.AllowedValues, v) {
			problems = append(problems, fmt.Sprintf("%s (%s): %q is not an allowed value (allowed: %s)", f.Name, id, v, strings.Join(f.AllowedValues, ", ")))
		}
	}
	if len(problems) > 0 {
		return fail(strings.Join(problems, "; "), "fix [issue_defaults.custom_fields]; 'jiractl inspect <issue>' shows IDs and values")
	}

	var missing []string
	for _, f := range fields {
		_, covered := d.CustomFields[f.ID]
		switch f.ID {
		case "components":
			covered = covered || d.Component != ""
		case "assignee":
			covered = covered || d.Assignee != ""
		case "labels":
			covered = covered || len(d.Labels) > 0
		}
		if f.Required && !f.HasDefault && !covered && !standardFields[f.ID] {
			missing = append(missing, fmt.Sprintf("%s (%s)", f.Name, f.ID))
		}
	}
	if len(missing) > 0 {
		return warn("required fields without a default: "+strings.Join(missing, ", "),
			"add them to [issue_defaults.custom_fields], or pass -F on create")
	}
	if len(ids) == 0 {
		return ok("none set")
	}
	return ok(fmt.Sprintf("%d valid", len(ids)))
}

// configuredOption extracts the option from {"value": ...} / {"name": ...}
// or a plain string.
func configuredOption(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "{") {
		var v struct{ Value, Name string }
		if json.Unmarshal([]byte(trimmed), &v) == nil {
			if v.Value != "" {
				return v.Value
			}
			return v.Name
		}
		return ""
	}
	if strings.HasPrefix(trimmed, "[") {
		return ""
	}
	return trimmed
}

func checkQueries(ctx *Ctx) Result {
	qs := ctx.Cfg.Queries
	if len(qs) == 0 {
		return ok("no saved queries")
	}
	jqls := make([]string, len(qs))
	for i, q := range qs {
		jqls[i] = ctx.Cfg.ExpandJQL(q.JQL)
	}
	errs, err := ctx.Client.ParseJQL(jqls)
	if err != nil {
		return warn("could not validate JQL: "+err.Error(), "")
	}
	var bad []string
	for i, e := range errs {
		if len(e) > 0 {
			bad = append(bad, fmt.Sprintf("%s: %s", qs[i].Name, strings.Join(e, " ")))
		}
	}
	if len(bad) > 0 {
		return fail(strings.Join(bad, "; "), "fix the jql of these [[queries]] in ~/"+config.ConfigFileName)
	}
	return ok(fmt.Sprintf("%d valid", len(qs)))
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// closest returns the candidate nearest to s (case-insensitive edit distance
// up to 3), or "".
func closest(s string, candidates []string) string {
	best, bestDist := "", 4
	for _, c := range candidates {
		if d := levenshtein(strings.ToLower(s), strings.ToLower(c)); d < bestDist {
			best, bestDist = c, d
		}
	}
	return best
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}
