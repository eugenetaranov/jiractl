package jira

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/keyring"
	"github.com/trivago/tgo/tcontainer"
)

type Client struct {
	*jira.Client
	config *config.Config
}

// SearchResult represents the response from the v3 search API
type SearchResult struct {
	Issues []jira.Issue `json:"issues"`
	Total  int          `json:"total"`
}

// NewClient creates a new Jira client using credentials from keyring and config
func NewClient(cfg *config.Config) (*Client, error) {
	username, token, err := keyring.GetCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	if username == "" || token == "" {
		return nil, fmt.Errorf("credentials not configured, run 'jiractl configure' first")
	}

	if cfg.Server == "" {
		return nil, fmt.Errorf("server URL not configured, run 'jiractl configure' first")
	}

	tp := jira.BasicAuthTransport{
		Username: strings.TrimSpace(username),
		Password: strings.TrimSpace(token),
	}

	client, err := jira.NewClient(tp.Client(), cfg.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	return &Client{
		Client: client,
		config: cfg,
	}, nil
}

// CreateIssueOptions contains optional fields for issue creation
type CreateIssueOptions struct {
	EpicLink string
}

// parseCustomFieldValue interprets a config string as JSON when it looks like
// a JSON object/array, otherwise returns it as a plain string. This lets users
// write select-list values as '{"value":"Operations"}' in TOML while still
// supporting simple string fields.
func parseCustomFieldValue(raw string) interface{} {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var v interface{}
		if err := json.Unmarshal([]byte(trimmed), &v); err == nil {
			return v
		}
	}
	return raw
}

// CreateIssue creates a new issue in Jira
func (c *Client) CreateIssue(project, issueType, summary, description string, opts *CreateIssueOptions) (*jira.Issue, error) {
	issue := &jira.Issue{
		Fields: &jira.IssueFields{
			Project: jira.Project{
				Key: project,
			},
			Type: jira.IssueType{
				Name: issueType,
			},
			Summary:     summary,
			Description: description,
		},
	}

	// Apply defaults from config
	if c.config.IssueDefaults.Assignee != "" && issue.Fields.Assignee == nil {
		issue.Fields.Assignee = &jira.User{Name: c.config.IssueDefaults.Assignee}
	}
	if len(c.config.IssueDefaults.Labels) > 0 && len(issue.Fields.Labels) == 0 {
		issue.Fields.Labels = c.config.IssueDefaults.Labels
	}

	// Apply epic link if provided
	epicLink := ""
	if opts != nil && opts.EpicLink != "" {
		epicLink = opts.EpicLink
	} else if c.config.IssueDefaults.EpicLink != "" {
		epicLink = c.config.IssueDefaults.EpicLink
	}

	if epicLink != "" {
		// Epic Link is typically a custom field. In Jira Cloud, it's often "parent" for next-gen projects
		// or a custom field like "customfield_10014" for classic projects.
		// We'll use the parent field which works for next-gen/team-managed projects.
		issue.Fields.Parent = &jira.Parent{Key: epicLink}
	}

	if len(c.config.IssueDefaults.CustomFields) > 0 {
		issue.Fields.Unknowns = tcontainer.MarshalMap{}
		for k, raw := range c.config.IssueDefaults.CustomFields {
			issue.Fields.Unknowns[k] = parseCustomFieldValue(raw)
		}
	}

	created, resp, err := c.Issue.Create(issue)
	if err != nil {
		if resp != nil && resp.Body != nil {
			body, _ := io.ReadAll(resp.Body)
			if len(body) > 0 {
				return nil, fmt.Errorf("failed to create issue: %s", string(body))
			}
			return nil, fmt.Errorf("failed to create issue (status %d): %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	return created, nil
}

// SearchIssues searches for issues using JQL via the v3 API
func (c *Client) SearchIssues(jql string, maxResults int) ([]jira.Issue, error) {
	if maxResults <= 0 {
		maxResults = 50
	}

	// Use the v3 search/jql endpoint
	apiEndpoint := fmt.Sprintf(
		"rest/api/3/search/jql?jql=%s&maxResults=%d&fields=key,summary,status,assignee,priority,created,updated",
		url.QueryEscape(jql),
		maxResults,
	)

	req, err := c.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Do(req, nil)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("search failed (status %d): %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer resp.Body.Close()

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Issues, nil
}

// GetIssue retrieves a single issue by key
func (c *Client) GetIssue(key string) (*jira.Issue, error) {
	issue, resp, err := c.Issue.Get(key, nil)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("failed to get issue (status %d): %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}
	return issue, nil
}

// RawIssue holds the decoded JSON of an issue together with the field-name
// and schema maps returned by the `expand=names,schema` query parameter.
type RawIssue struct {
	Key    string                 `json:"key"`
	Fields map[string]interface{} `json:"fields"`
	Names  map[string]string      `json:"names"`
	Schema map[string]interface{} `json:"schema"`
}

// GetIssueRaw fetches an issue with expand=names,schema so callers can see
// every field (including customfield_*) with its human-readable name. Returns
// both the decoded struct and the pretty-printed raw JSON body.
func (c *Client) GetIssueRaw(key string, debug bool) (*RawIssue, []byte, error) {
	apiEndpoint := fmt.Sprintf("rest/api/2/issue/%s?expand=names,schema", url.PathEscape(key))

	req, err := c.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "DEBUG: GET %s\n", req.URL.String())
	}

	resp, err := c.Do(req, nil)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			return nil, nil, fmt.Errorf("failed to get issue (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, nil, fmt.Errorf("failed to get issue: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	var raw RawIssue
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, body, fmt.Errorf("failed to decode response: %w", err)
	}

	pretty, err := json.MarshalIndent(json.RawMessage(body), "", "  ")
	if err != nil {
		pretty = body
	}

	return &raw, pretty, nil
}

// GetIssueTypes returns available issue types for a project
func (c *Client) GetIssueTypes(projectKey string) ([]jira.IssueType, error) {
	project, resp, err := c.Project.Get(projectKey)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("failed to get project (status %d): %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	return project.IssueTypes, nil
}

// TestConnection verifies the connection to Jira works
func (c *Client) TestConnection() error {
	_, resp, err := c.User.GetSelf()
	if err != nil {
		if resp != nil {
			return fmt.Errorf("connection test failed (status %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("connection test failed: %w", err)
	}
	return nil
}

// GetEpics returns open epics in the given project
func (c *Client) GetEpics(projectKey string) ([]jira.Issue, error) {
	jql := fmt.Sprintf("project = %s AND issuetype = Epic AND resolution = Unresolved ORDER BY created DESC", projectKey)
	return c.SearchIssues(jql, 100)
}
