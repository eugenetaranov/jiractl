package config

import (
	"strings"
	"testing"
)

func TestFindQuery(t *testing.T) {
	c := &Config{Queries: []Query{{Name: "mine"}, {Name: "My Team"}, {Name: "recent"}, {Name: "unassigned"}}}
	for in, want := range map[string]string{"mine": "mine", "MINE": "mine", "rec": "recent", "U": "unassigned", "my team": "My Team"} {
		q, err := c.FindQuery(in)
		if err != nil || q.Name != want {
			t.Errorf("%q: got %v %v, want %s", in, q, err, want)
		}
	}
	if _, err := c.FindQuery("m"); err == nil || !strings.Contains(err.Error(), "ambiguous: mine, My Team") {
		t.Errorf("ambiguous: %v", err)
	}
	if _, err := c.FindQuery("foo"); err == nil || err.Error() != `no query "foo"; available: mine, My Team, recent, unassigned` {
		t.Errorf("none: %v", err)
	}
	if _, err := (&Config{}).FindQuery("x"); err == nil || !strings.Contains(err.Error(), "no queries configured") {
		t.Errorf("empty: %v", err)
	}
}
