package jira

import (
	"sort"
	"time"

	jira "github.com/andygrunwald/go-jira"
)

// StatusRank orders status categories for display: In Progress (0), To Do
// (1), Done (2). Jira's category keys are stable across workflows, unlike
// status names ("Done", "Completed", "Closed"...).
func StatusRank(is jira.Issue) int {
	if is.Fields == nil {
		return 1
	}
	if is.Fields.Status != nil {
		switch is.Fields.Status.StatusCategory.Key {
		case "indeterminate":
			return 0
		case "new":
			return 1
		case "done":
			return 2
		}
	}
	if is.Fields.Resolution != nil {
		return 2
	}
	return 1
}

// UpdatedAt returns when the issue was last updated (zero if unknown).
func UpdatedAt(is jira.Issue) time.Time {
	if is.Fields == nil {
		return time.Time{}
	}
	return time.Time(is.Fields.Updated)
}

// SortForDisplay orders issues for people: In Progress, then To Do, then
// Done, most recently updated first within each group. Equal keys keep
// Jira's order.
func SortForDisplay(issues []jira.Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		ri, rj := StatusRank(issues[i]), StatusRank(issues[j])
		if ri != rj {
			return ri < rj
		}
		return UpdatedAt(issues[i]).After(UpdatedAt(issues[j]))
	})
}
