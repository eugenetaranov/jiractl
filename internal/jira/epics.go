package jira

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	jira "github.com/andygrunwald/go-jira"
)

// EpicState is the result of checking whether an epic can be used as parent.
type EpicState int

const (
	EpicOK EpicState = iota
	EpicNotFound
	EpicNotAnEpic
	EpicUnknown // lookup failed for another reason (network, permissions)
)

var issueKeyRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]+-\d+$`)

// CheckEpic reports whether key exists and is an epic. The issue is returned
// when it was found.
func (c *Client) CheckEpic(key string) (EpicState, *jira.Issue, error) {
	issue, err := c.GetIssue(key)
	if err != nil {
		if StatusOf(err) == 404 {
			return EpicNotFound, nil, nil
		}
		return EpicUnknown, nil, err
	}
	if issue.Fields == nil || !strings.EqualFold(issue.Fields.Type.Name, "Epic") {
		return EpicNotAnEpic, issue, nil
	}
	return EpicOK, issue, nil
}

// EpicSearchJQL builds the JQL for an epic search. Every word must match the
// summary as a prefix; input that looks like an issue key also matches the
// key. Empty text lists open epics.
func EpicSearchJQL(project, text string, wildcard bool) string {
	base := fmt.Sprintf("project = %s AND issuetype = Epic", project)
	text = strings.TrimSpace(text)
	if text == "" {
		return base + " AND resolution = Unresolved ORDER BY updated DESC"
	}

	var terms []string
	for _, w := range strings.Fields(text) {
		w = strings.NewReplacer(`"`, "", `\`, "", "'", "").Replace(w)
		if w == "" {
			continue
		}
		if wildcard {
			w += "*"
		}
		terms = append(terms, fmt.Sprintf(`summary ~ "%s"`, w))
	}
	cond := strings.Join(terms, " AND ")
	if issueKeyRE.MatchString(text) {
		key := fmt.Sprintf(`key = "%s"`, strings.ToUpper(text))
		if cond == "" {
			cond = key
		} else {
			cond = fmt.Sprintf("(%s) OR %s", cond, key)
		}
	}
	if cond == "" {
		return base + " ORDER BY updated DESC"
	}
	return fmt.Sprintf("%s AND (%s) ORDER BY updated DESC", base, cond)
}

// SearchEpics finds epics in project matching text, unresolved ones first.
func (c *Client) SearchEpics(project, text string) ([]jira.Issue, error) {
	issues, err := c.SearchIssues(EpicSearchJQL(project, text, true), 50)
	if err != nil && StatusOf(err) == 400 {
		// Some instances reject wildcard text search; retry with plain words.
		issues, err = c.SearchIssues(EpicSearchJQL(project, text, false), 50)
	}
	if err != nil {
		return nil, err
	}
	sort.SliceStable(issues, func(i, j int) bool {
		return !IsResolved(issues[i]) && IsResolved(issues[j])
	})
	return issues, nil
}

// IsResolved reports whether the issue has a resolution.
func IsResolved(issue jira.Issue) bool {
	return issue.Fields != nil && issue.Fields.Resolution != nil
}

// IsEpicRejection reports whether a create failed because of the parent or
// epic field.
func IsEpicRejection(err error) bool {
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	if apiErr.HasField("parent") {
		return true
	}
	for field, msg := range apiErr.Fields {
		if strings.Contains(strings.ToLower(field+" "+msg), "epic") {
			return true
		}
	}
	return false
}
