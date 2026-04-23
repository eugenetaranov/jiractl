package cmd

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/spf13/cobra"
)

var inspectRaw bool

var inspectCmd = &cobra.Command{
	Use:   "inspect <ticket-url-or-key>",
	Short: "Inspect a Jira issue and show all fields",
	Long: `Fetch a Jira issue and print every field including custom fields with their IDs
and human-readable names. Useful for discovering what to put in config (e.g. which
customfield_XXXXX id maps to "Team" or "Component" in your project).

Accepts a full browse URL or an issue key:
  jiractl inspect https://example.atlassian.net/browse/PROJ-123
  jiractl inspect PROJ-123`,
	Args: cobra.ExactArgs(1),
	RunE: runInspect,
}

func init() {
	inspectCmd.Flags().BoolVar(&inspectRaw, "raw", false, "Print the raw JSON response")
	RootCmd.AddCommand(inspectCmd)
}

var issueKeyRE = regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

func parseIssueKey(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty input")
	}
	match := issueKeyRE.FindString(strings.ToUpper(input))
	if match == "" {
		return "", fmt.Errorf("could not extract issue key from %q", input)
	}
	return match, nil
}

func runInspect(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	key, err := parseIssueKey(args[0])
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client, err := jira.NewClient(cfg)
	if err != nil {
		return err
	}

	raw, body, err := client.GetIssueRaw(key)
	if err != nil {
		return err
	}

	if inspectRaw {
		fmt.Println(string(body))
		return nil
	}

	printInspect(raw)
	return nil
}

func printInspect(raw *jira.RawIssue) {
	fmt.Printf("Issue: %s\n", raw.Key)
	fmt.Println(strings.Repeat("─", 60))

	// Core fields first
	core := []string{"project", "issuetype", "status", "priority", "assignee", "reporter", "summary", "labels", "parent"}
	for _, name := range core {
		if v, ok := raw.Fields[name]; ok {
			fmt.Printf("%-20s %s\n", name+":", formatValue(v))
		}
	}

	// All other non-custom fields
	fmt.Println("\nStandard fields:")
	var stdKeys []string
	for k := range raw.Fields {
		if strings.HasPrefix(k, "customfield_") {
			continue
		}
		if contains(core, k) {
			continue
		}
		stdKeys = append(stdKeys, k)
	}
	sort.Strings(stdKeys)
	for _, k := range stdKeys {
		v := raw.Fields[k]
		if isEmpty(v) {
			continue
		}
		fmt.Printf("  %-28s %s\n", k, formatValue(v))
	}

	// Custom fields with resolved names — this is the useful bit for config
	fmt.Println("\nCustom fields (use these IDs in [issue_defaults.custom_fields]):")
	var cfKeys []string
	for k := range raw.Fields {
		if strings.HasPrefix(k, "customfield_") {
			cfKeys = append(cfKeys, k)
		}
	}
	sort.Strings(cfKeys)
	for _, k := range cfKeys {
		v := raw.Fields[k]
		name := raw.Names[k]
		marker := ""
		if isEmpty(v) {
			marker = " (empty)"
		}
		fmt.Printf("  %s  %q%s\n", k, name, marker)
		if !isEmpty(v) {
			fmt.Printf("      = %s\n", formatValue(v))
		}
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func isEmpty(v interface{}) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case []interface{}:
		return len(x) == 0
	case map[string]interface{}:
		return len(x) == 0
	}
	return false
}

// formatValue renders a field value compactly. Objects with common shape
// ({"name":...}, {"value":...}, {"key":...,"name":...}) are collapsed to
// their identifying field; anything else falls back to compact JSON.
func formatValue(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return "<nil>"
	case string:
		return x
	case bool:
		return fmt.Sprintf("%v", x)
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%v", x)
	case []interface{}:
		parts := make([]string, len(x))
		for i, item := range x {
			parts[i] = formatValue(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]interface{}:
		// Common Jira shape: object with a display-ish field
		for _, k := range []string{"displayName", "name", "value", "key"} {
			if s, ok := x[k].(string); ok && s != "" {
				if kk, ok := x["key"].(string); ok && k != "key" && kk != "" {
					return fmt.Sprintf("%s (%s)", s, kk)
				}
				return s
			}
		}
		b, _ := json.Marshal(x)
		return string(b)
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}
