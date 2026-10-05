package config

import (
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// patchConfig edits the TOML text in place so it matches desired, changing
// only the lines for keys whose values differ. It returns ok=false when the
// file can't be patched safely; the caller then falls back to a full rewrite.
func patchConfig(text string, desired *Config) (string, bool) {
	var current Config
	if _, err := toml.Decode(text, &current); err != nil {
		return "", false
	}

	eol := "\n"
	if strings.Contains(text, "\r\n") {
		eol = "\r\n"
	}
	p := &patcher{lines: strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")}

	type change struct {
		table, key string
		value      interface{} // nil means remove the key
	}
	var changes []change
	str := func(table, key, cur, want string) {
		if cur == want {
			return
		}
		if want == "" {
			changes = append(changes, change{table, key, nil})
		} else {
			changes = append(changes, change{table, key, want})
		}
	}

	str("", "server", current.Server, desired.Server)
	str("", "project", current.Project, desired.Project)
	cd, dd := current.IssueDefaults, desired.IssueDefaults
	str("issue_defaults", "assignee", cd.Assignee, dd.Assignee)
	str("issue_defaults", "component", cd.Component, dd.Component)
	str("issue_defaults", "epic_link", cd.EpicLink, dd.EpicLink)
	str("issue_defaults", "issue_type", cd.IssueType, dd.IssueType)
	if !equalStrings(cd.Labels, dd.Labels) {
		if len(dd.Labels) == 0 {
			changes = append(changes, change{"issue_defaults", "labels", nil})
		} else {
			changes = append(changes, change{"issue_defaults", "labels", dd.Labels})
		}
	}

	cfKeys := map[string]bool{}
	for k := range cd.CustomFields {
		cfKeys[k] = true
	}
	for k := range dd.CustomFields {
		cfKeys[k] = true
	}
	sortedCF := make([]string, 0, len(cfKeys))
	for k := range cfKeys {
		sortedCF = append(sortedCF, k)
	}
	sort.Strings(sortedCF)
	for _, k := range sortedCF {
		cur, curOK := cd.CustomFields[k]
		want, wantOK := dd.CustomFields[k]
		switch {
		case !wantOK && curOK:
			changes = append(changes, change{"issue_defaults.custom_fields", k, nil})
		case wantOK && (!curOK || cur != want):
			changes = append(changes, change{"issue_defaults.custom_fields", k, want})
		}
	}

	// Queries can only be appended; anything else needs a rewrite.
	if len(desired.Queries) < len(current.Queries) {
		return "", false
	}
	for i := range current.Queries {
		if current.Queries[i] != desired.Queries[i] {
			return "", false
		}
	}

	for _, c := range changes {
		if !p.set(c.table, c.key, c.value) {
			return "", false
		}
	}
	for _, q := range desired.Queries[len(current.Queries):] {
		block := []string{"[[queries]]", "name = " + literal(q.Name), "jql = " + literal(q.JQL)}
		if q.Limit > 0 {
			block = append(block, "limit = "+literal(q.Limit))
		}
		p.appendBlock(block)
	}

	out := strings.Join(p.lines, "\n")
	if eol != "\n" {
		out = strings.ReplaceAll(out, "\n", eol)
	}

	// Never trust the patch blindly: it must decode to exactly what we want.
	var check Config
	if _, err := toml.Decode(out, &check); err != nil {
		return "", false
	}
	if !reflect.DeepEqual(normalize(check), normalize(*desired)) {
		return "", false
	}
	return out, true
}

type patcher struct {
	lines []string
}

var bareKeyRE = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// tableOf returns the table each line belongs to. Array-of-tables sections
// are reported as "[[name]]" so they never match a regular table.
func (p *patcher) tableOf() []string {
	tables := make([]string, len(p.lines))
	current := ""
	for i, line := range p.lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[[") {
			if end := strings.Index(t, "]]"); end > 0 {
				current = "[[" + normalizeTable(t[2:end]) + "]]"
			}
		} else if strings.HasPrefix(t, "[") {
			if end := strings.Index(t, "]"); end > 0 {
				current = normalizeTable(t[1:end])
			}
		}
		tables[i] = current
	}
	return tables
}

func normalizeTable(name string) string {
	parts := strings.Split(name, ".")
	for i, part := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(part), `"'`)
	}
	return strings.Join(parts, ".")
}

func isHeader(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "[")
}

func isKeyLine(line string) bool {
	t := strings.TrimSpace(line)
	return t != "" && !strings.HasPrefix(t, "#") && !strings.HasPrefix(t, "[")
}

func lineKey(line string) string {
	t := strings.TrimSpace(line)
	eq := strings.Index(t, "=")
	if eq < 0 {
		return ""
	}
	return strings.Trim(strings.TrimSpace(t[:eq]), `"'`)
}

func (p *patcher) set(table, key string, value interface{}) bool {
	tables := p.tableOf()
	for i, line := range p.lines {
		if tables[i] != table || isHeader(line) || !isKeyLine(line) || lineKey(line) != key {
			continue
		}
		if value == nil {
			p.lines = append(p.lines[:i], p.lines[i+1:]...)
			return true
		}
		replaced, ok := replaceValue(line, literal(value))
		if !ok {
			return false
		}
		p.lines[i] = replaced
		return true
	}

	if value == nil {
		return true
	}
	newLine := keyLiteral(key) + " = " + literal(value)

	// Insert after the last key line of the table, or after its header.
	insertAt := -1
	headerFound := false
	for i, line := range p.lines {
		if tables[i] != table {
			continue
		}
		if isHeader(line) {
			headerFound = true
			insertAt = i + 1
		} else if isKeyLine(line) {
			insertAt = i + 1
		}
	}
	if table == "" && insertAt < 0 {
		insertAt = 0
	}
	if insertAt >= 0 && (table == "" || headerFound) {
		p.lines = append(p.lines[:insertAt], append([]string{newLine}, p.lines[insertAt:]...)...)
		return true
	}

	p.appendBlock([]string{"[" + table + "]", newLine})
	return true
}

// appendBlock adds lines at the end of the file, separated by a blank line.
func (p *patcher) appendBlock(block []string) {
	for len(p.lines) > 0 && strings.TrimSpace(p.lines[len(p.lines)-1]) == "" {
		p.lines = p.lines[:len(p.lines)-1]
	}
	if len(p.lines) > 0 {
		p.lines = append(p.lines, "")
	}
	p.lines = append(p.lines, block...)
	p.lines = append(p.lines, "")
}

// replaceValue swaps the value of a `key = value  # comment` line, keeping
// indentation, key spelling and any trailing comment.
func replaceValue(line, lit string) (string, bool) {
	eq := strings.Index(line, "=")
	if eq < 0 {
		return "", false
	}
	rest := line[eq+1:]
	value := strings.TrimSpace(rest)
	if strings.HasPrefix(value, `"""`) || strings.HasPrefix(value, `'''`) || strings.HasPrefix(value, "{") {
		return "", false
	}

	comment := ""
	if idx := commentIndex(rest); idx >= 0 {
		comment = strings.TrimSpace(rest[idx:])
		value = strings.TrimSpace(rest[:idx])
	}
	if strings.HasPrefix(value, "[") && !strings.HasSuffix(value, "]") {
		return "", false // multi-line array
	}

	out := strings.TrimRight(line[:eq], " \t") + " = " + lit
	if comment != "" {
		out += " " + comment
	}
	return out, true
}

// commentIndex finds the start of a trailing comment, ignoring '#' inside
// quoted strings.
func commentIndex(s string) int {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote == '"' && c == '\\':
			i++
		case quote != 0 && c == quote:
			quote = 0
		case quote == 0 && (c == '"' || c == '\''):
			quote = c
		case quote == 0 && c == '#':
			return i
		}
	}
	return -1
}

func keyLiteral(key string) string {
	if bareKeyRE.MatchString(key) {
		return key
	}
	return literal(key)
}

// literal renders a value as TOML using the real encoder.
func literal(v interface{}) string {
	var sb strings.Builder
	if err := toml.NewEncoder(&sb).Encode(map[string]interface{}{"v": v}); err != nil {
		return `""`
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(sb.String()), "v = "))
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func normalize(c Config) Config {
	if len(c.IssueDefaults.Labels) == 0 {
		c.IssueDefaults.Labels = nil
	}
	if len(c.IssueDefaults.CustomFields) == 0 {
		c.IssueDefaults.CustomFields = nil
	}
	if len(c.Queries) == 0 {
		c.Queries = nil
	}
	return c
}
