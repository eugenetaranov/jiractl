package config

import (
	"fmt"
	"strings"
)

// StarterQueries are added by configure when the config has no queries.
var StarterQueries = []Query{
	{Name: "mine", JQL: "project = ${project} AND assignee = currentUser() AND statusCategory != Done ORDER BY updated DESC", Limit: 50},
	{Name: "recent", JQL: "project = ${project} AND updated >= -7d ORDER BY updated DESC", Limit: 50},
	{Name: "unassigned", JQL: "project = ${project} AND assignee is EMPTY AND statusCategory != Done ORDER BY created DESC", Limit: 50},
}

// FindQuery matches a query name exactly, then case-insensitively, then by
// unique case-insensitive prefix. Errors list the candidates.
func (c *Config) FindQuery(name string) (*Query, error) {
	if len(c.Queries) == 0 {
		return nil, fmt.Errorf("no queries configured; run 'jiractl configure' to add starter queries or add [[queries]] to ~/%s", ConfigFileName)
	}
	if q := c.GetQuery(name); q != nil {
		return q, nil
	}

	lower := strings.ToLower(name)
	var folded, prefixed []*Query
	for i := range c.Queries {
		q := &c.Queries[i]
		qn := strings.ToLower(q.Name)
		if qn == lower {
			folded = append(folded, q)
		}
		if strings.HasPrefix(qn, lower) {
			prefixed = append(prefixed, q)
		}
	}

	for _, matches := range [][]*Query{folded, prefixed} {
		switch len(matches) {
		case 0:
			continue
		case 1:
			return matches[0], nil
		default:
			names := make([]string, len(matches))
			for i, q := range matches {
				names[i] = q.Name
			}
			return nil, fmt.Errorf("query %q is ambiguous: %s", name, strings.Join(names, ", "))
		}
	}
	return nil, fmt.Errorf("no query %q; available: %s", name, strings.Join(c.QueryNames(), ", "))
}
