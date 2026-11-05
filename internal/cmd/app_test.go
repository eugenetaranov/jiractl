package cmd

import (
	"strings"
	"testing"

	"github.com/eugenetaranov/jiractl/internal/config"
)

func TestQueryPickerListsSearchLast(t *testing.T) {
	cfg := &config.Config{Queries: []config.Query{{Name: "mine"}, {Name: "recent"}}}
	view := newQueryPickScreen(cfg).View(80, 20)
	mine, recent, search := strings.Index(view, "mine"), strings.Index(view, "recent"), strings.Index(view, searchIssuesRow)
	if mine < 0 || recent < mine || search < recent {
		t.Fatalf("want mine, recent, then %q:\n%s", searchIssuesRow, view)
	}

	// With no saved queries the search row is still there.
	if view := newQueryPickScreen(&config.Config{}).View(80, 20); !strings.Contains(view, searchIssuesRow) {
		t.Fatalf("search row missing:\n%s", view)
	}
}
