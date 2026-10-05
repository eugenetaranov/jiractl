package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/eugenetaranov/jiractl/internal/jira"
)

// fieldNames maps Jira field IDs to display names and back.
type fieldNames struct {
	byID   map[string]string
	byName map[string]string // lower-case name -> ID
}

func newFieldNames(fields []jira.Field) *fieldNames {
	f := &fieldNames{byID: map[string]string{}, byName: map[string]string{}}
	for _, field := range fields {
		f.byID[field.ID] = field.Name
		f.byName[strings.ToLower(field.Name)] = field.ID
	}
	return f
}

// label returns the display name of a field, or its ID when unknown.
func (f *fieldNames) label(id string) string {
	if f != nil {
		if name, ok := f.byID[id]; ok && name != "" {
			return name
		}
	}
	return id
}

// resolve turns a field name or ID into an ID.
func (f *fieldNames) resolve(key string) (string, error) {
	key = strings.TrimSpace(key)
	if f == nil {
		if strings.HasPrefix(key, "customfield_") {
			return key, nil
		}
		return "", fmt.Errorf("cannot resolve field %q: field names are unavailable; use its customfield_ ID", key)
	}
	if _, ok := f.byID[key]; ok {
		return key, nil
	}
	if id, ok := f.byName[strings.ToLower(key)]; ok {
		return id, nil
	}
	if strings.HasPrefix(key, "customfield_") {
		return key, nil
	}
	msg := fmt.Sprintf("unknown field %q", key)
	if close := f.suggest(key); len(close) > 0 {
		msg += "; did you mean: " + strings.Join(close, ", ")
	}
	return "", fmt.Errorf("%s", msg)
}

func (f *fieldNames) suggest(key string) []string {
	key = strings.ToLower(key)
	type cand struct {
		name string
		dist int
	}
	var cands []cand
	for id, name := range f.byID {
		lower := strings.ToLower(name)
		d := levenshtein(key, lower)
		if strings.Contains(lower, key) || strings.Contains(key, lower) {
			d = 0
		}
		if d <= 3 {
			cands = append(cands, cand{fmt.Sprintf("%s (%s)", name, id), d})
		}
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].dist != cands[j].dist {
			return cands[i].dist < cands[j].dist
		}
		return cands[i].name < cands[j].name
	})
	var out []string
	for i := 0; i < len(cands) && i < 5; i++ {
		out = append(out, cands[i].name)
	}
	return out
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

// parseFieldFlag splits a -F key=value flag.
func parseFieldFlag(s string) (string, string, error) {
	k, v, ok := strings.Cut(s, "=")
	if !ok || strings.TrimSpace(k) == "" {
		return "", "", fmt.Errorf("invalid -F %q: expected name=value", s)
	}
	return strings.TrimSpace(k), v, nil
}
