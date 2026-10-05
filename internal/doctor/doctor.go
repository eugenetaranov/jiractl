// Package doctor checks whether jiractl is ready to create issues and run
// queries: config, credentials, server, permissions, defaults and saved JQL.
package doctor

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

type Status int

const (
	OK Status = iota
	Warn
	Fail
	Skip
)

func (s Status) String() string {
	return [...]string{"ok", "warn", "fail", "skip"}[s]
}

func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// Result is the outcome of one check.
type Result struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status Status `json:"status"`
	Detail string `json:"detail,omitempty"`
	Hint   string `json:"hint,omitempty"`
}

// Check is one readiness check. Checks run in order; a check whose
// dependency didn't pass (OK or Warn) is skipped. Consecutive Parallel checks
// run concurrently, so they may only depend on earlier checks.
type Check struct {
	ID        string
	Title     string
	DependsOn []string
	Parallel  bool
	Run       func(*Ctx) Result
}

func ok(detail string) Result { return Result{Status: OK, Detail: detail} }

func warn(detail, hint string) Result { return Result{Status: Warn, Detail: detail, Hint: hint} }

func fail(detail, hint string) Result { return Result{Status: Fail, Detail: detail, Hint: hint} }

// Run executes checks and returns their results in order.
func Run(ctx *Ctx, checks []Check) []Result {
	results := make([]Result, len(checks))
	byID := map[string]Result{}
	titles := map[string]string{}
	for _, c := range checks {
		titles[c.ID] = c.Title
	}

	run := func(i int) Result {
		c := checks[i]
		for _, dep := range c.DependsOn {
			if r, ok := byID[dep]; ok && r.Status != OK && r.Status != Warn {
				return Result{Status: Skip, Detail: "requires " + titles[dep]}
			}
		}
		return c.Run(ctx)
	}
	finish := func(i int, r Result) {
		r.ID, r.Title = checks[i].ID, checks[i].Title
		results[i] = r
	}

	for i := 0; i < len(checks); {
		if !checks[i].Parallel {
			finish(i, run(i))
			byID[checks[i].ID] = results[i]
			i++
			continue
		}
		j := i
		for j < len(checks) && checks[j].Parallel {
			j++
		}
		var wg sync.WaitGroup
		for k := i; k < j; k++ {
			wg.Add(1)
			go func(k int) {
				defer wg.Done()
				finish(k, run(k))
			}(k)
		}
		wg.Wait()
		for k := i; k < j; k++ {
			byID[checks[k].ID] = results[k]
		}
		i = j
	}
	return results
}

// Failed reports whether any check failed.
func Failed(results []Result) bool {
	for _, r := range results {
		if r.Status == Fail {
			return true
		}
	}
	return false
}

// Print writes results for people. fancy selects ✓/!/✗ markers and color;
// otherwise plain ASCII markers are used.
func Print(w io.Writer, results []Result, fancy bool) error {
	width := 0
	for _, r := range results {
		width = max(width, len(r.Title))
	}
	var sb strings.Builder
	for _, r := range results {
		marker := "[" + r.Status.String() + "]"
		if fancy {
			marker = [...]string{"\033[32m✓\033[0m", "\033[33m!\033[0m", "\033[31m✗\033[0m", "\033[2m–\033[0m"}[r.Status]
		} else {
			marker = fmt.Sprintf("%-6s", marker)
		}
		line := fmt.Sprintf("%s %-*s", marker, width, r.Title)
		if r.Detail != "" {
			line += "  " + r.Detail
		}
		sb.WriteString(strings.TrimRight(line, " ") + "\n")
		if r.Hint != "" && (r.Status == Warn || r.Status == Fail) {
			sb.WriteString("  → " + r.Hint + "\n")
		}
	}
	counts := map[Status]int{}
	for _, r := range results {
		counts[r.Status]++
	}
	fmt.Fprintf(&sb, "\n%d ok, %d warnings, %d failed, %d skipped\n", counts[OK], counts[Warn], counts[Fail], counts[Skip])
	_, err := io.WriteString(w, sb.String())
	return err
}

// PrintJSON writes results as a JSON array.
func PrintJSON(w io.Writer, results []Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}
