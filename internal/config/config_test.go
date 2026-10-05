package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

const sample = `# jiractl config
server = "https://old.atlassian.net"  # main instance
project = "OLD"

# team board
[issue_defaults]
issue_type = "Task"
labels = ["a", "b"]

[issue_defaults.custom_fields]
customfield_1 = '{"value": "Ops"}'

# mine
[[queries]]
name = "Mine"
jql = "assignee = currentUser()"
`

func decode(t *testing.T, s string) Config {
	t.Helper()
	var c Config
	if _, err := toml.Decode(s, &c); err != nil {
		t.Fatalf("decode: %v\n%s", err, s)
	}
	return c
}

func diffLines(a, b string) []string {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	set := map[string]bool{}
	for _, l := range al {
		set[l] = true
	}
	var out []string
	for _, l := range bl {
		if !set[l] {
			out = append(out, l)
		}
	}
	return out
}

func TestPatchChangesOnlyProject(t *testing.T) {
	want := decode(t, sample)
	want.Project = "NEW"
	out, ok := patchConfig(sample, &want)
	if !ok {
		t.Fatal("patch failed")
	}
	if !strings.Contains(out, "# team board") || !strings.Contains(out, "# main instance") {
		t.Fatalf("comments lost:\n%s", out)
	}
	if d := diffLines(sample, out); len(d) != 1 || d[0] != `project = "NEW"` {
		t.Fatalf("unexpected diff %q", d)
	}
}

func TestPatchKeepsTrailingComment(t *testing.T) {
	want := decode(t, sample)
	want.Server = "https://new.atlassian.net"
	out, ok := patchConfig(sample, &want)
	if !ok {
		t.Fatal("patch failed")
	}
	if !strings.Contains(out, `server = "https://new.atlassian.net" # main instance`) {
		t.Fatalf("server line wrong:\n%s", out)
	}
}

func TestPatchInsertsMissingKeyUnderTable(t *testing.T) {
	want := decode(t, sample)
	want.IssueDefaults.EpicLink = "OPS-40"
	out, ok := patchConfig(sample, &want)
	if !ok {
		t.Fatal("patch failed")
	}
	got := decode(t, out)
	if got.IssueDefaults.EpicLink != "OPS-40" {
		t.Fatalf("epic not set:\n%s", out)
	}
	idx := strings.Index(out, "epic_link")
	if idx < strings.Index(out, "[issue_defaults]") || idx > strings.Index(out, "[issue_defaults.custom_fields]") {
		t.Fatalf("epic_link inserted in wrong place:\n%s", out)
	}
}

func TestPatchRemovesKey(t *testing.T) {
	want := decode(t, sample)
	want.IssueDefaults.IssueType = ""
	out, ok := patchConfig(sample, &want)
	if !ok {
		t.Fatal("patch failed")
	}
	if strings.Contains(out, "issue_type") {
		t.Fatalf("issue_type not removed:\n%s", out)
	}
}

func TestPatchCreatesMissingTable(t *testing.T) {
	src := "server = \"https://x\"\nproject = \"P\"\n"
	want := decode(t, src)
	want.IssueDefaults.IssueType = "Bug"
	out, ok := patchConfig(src, &want)
	if !ok {
		t.Fatal("patch failed")
	}
	if !strings.Contains(out, "[issue_defaults]\nissue_type = \"Bug\"") {
		t.Fatalf("table not created:\n%s", out)
	}
}

func TestPatchAppendsQueries(t *testing.T) {
	want := decode(t, sample)
	want.Queries = append(want.Queries, Query{Name: "recent", JQL: "updated >= -7d", Limit: 20})
	out, ok := patchConfig(sample, &want)
	if !ok {
		t.Fatal("patch failed")
	}
	if got := decode(t, out); len(got.Queries) != 2 || got.Queries[1].Limit != 20 {
		t.Fatalf("query not appended:\n%s", out)
	}
	if !strings.Contains(out, "# mine") {
		t.Fatal("comment lost")
	}
}

func TestPatchCRLF(t *testing.T) {
	src := strings.ReplaceAll(sample, "\n", "\r\n")
	want := decode(t, src)
	want.Project = "NEW"
	out, ok := patchConfig(src, &want)
	if !ok {
		t.Fatal("patch failed")
	}
	if strings.Count(out, "\r\n") != strings.Count(src, "\r\n") || strings.Contains(strings.ReplaceAll(out, "\r\n", ""), "\n") {
		t.Fatalf("line endings changed:\n%q", out)
	}
}

func TestPatchRefusesInlineTable(t *testing.T) {
	src := "server = \"https://x\"\nproject = \"P\"\nissue_defaults = { issue_type = \"Task\" }\n"
	want := decode(t, src)
	want.IssueDefaults.IssueType = "Bug"
	if _, ok := patchConfig(src, &want); ok {
		t.Fatal("expected patch to refuse inline table")
	}
}

func TestSaveFallbackAndPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFileName)
	src := "server = \"https://x\"\nproject = \"P\"\nissue_defaults = { issue_type = \"Task\" }\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	want := decode(t, src)
	want.IssueDefaults.IssueType = "Bug"
	if err := want.saveTo(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
	if bak, err := os.ReadFile(path + ".bak"); err != nil || string(bak) != src {
		t.Fatalf("backup missing or wrong: %v", err)
	}
	data, _ := os.ReadFile(path)
	if decode(t, string(data)).IssueDefaults.IssueType != "Bug" {
		t.Fatal("value not saved")
	}
}

func TestWriteFailureLeavesOriginal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, []byte(sample), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0o700) }()

	want := decode(t, sample)
	want.Project = "NEW"
	if err := want.saveTo(path); err == nil {
		t.Fatal("expected write error in read-only dir")
	}
	data, _ := os.ReadFile(path)
	if string(data) != sample {
		t.Fatal("original changed")
	}
}

func TestUnknownKeysWarn(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ConfigFileName)
	if err := os.WriteFile(path, []byte("server = \"x\"\nproject = \"P\"\n[issue_defaults]\nasignee = \"bob\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	r, w, _ := os.Pipe()
	old := os.Stderr
	os.Stderr = w
	_, err := Load()
	_ = w.Close()
	os.Stderr = old
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	if !strings.Contains(string(buf[:n]), `unknown config key "issue_defaults.asignee"`) {
		t.Fatalf("no warning: %q", buf[:n])
	}
}

func TestPatchDeploymentAndEpicField(t *testing.T) {
	want := decode(t, sample)
	want.Deployment = DeploymentServer
	want.IssueDefaults.EpicField = "customfield_10014"
	out, ok := patchConfig(sample, &want)
	if !ok {
		t.Fatal("patch failed")
	}
	got := decode(t, out)
	if !got.IsServer() || got.IssueDefaults.EpicField != "customfield_10014" || !strings.Contains(out, "# team board") {
		t.Fatalf("got:\n%s", out)
	}
}
