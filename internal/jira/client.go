package jira

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	jira "github.com/andygrunwald/go-jira"
	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/keyring"
	"github.com/trivago/tgo/tcontainer"
)

type Client struct {
	*jira.Client
	config *config.Config

	accountMu sync.Mutex
	accountID map[string]string // assignee value -> resolved account ID
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

	return NewClientWith(cfg, username, token)
}

// NewClientWith creates a client from explicit credentials, so values can be
// tested before anything is written to the keyring or config file.
func NewClientWith(cfg *config.Config, username, token string) (*Client, error) {
	tp := jira.BasicAuthTransport{
		Username: strings.TrimSpace(username),
		Password: strings.TrimSpace(token),
	}

	client, err := jira.NewClient(tp.Client(), cfg.Server)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	return &Client{
		Client:    client,
		config:    cfg,
		accountID: map[string]string{},
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
	var assignee map[string]string
	if c.config.IssueDefaults.Assignee != "" {
		accountID, err := c.ResolveAccountID(c.config.IssueDefaults.Assignee)
		if err != nil {
			return nil, err
		}
		// Sent as a raw field: go-jira's User type always serializes an
		// empty Password, which has no place in a create request.
		assignee = map[string]string{"accountId": accountID}
	}
	if c.config.IssueDefaults.Component != "" && len(issue.Fields.Components) == 0 {
		issue.Fields.Components = []*jira.Component{{Name: c.config.IssueDefaults.Component}}
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

	issue.Fields.Unknowns = tcontainer.MarshalMap{}
	for k, raw := range c.config.IssueDefaults.CustomFields {
		issue.Fields.Unknowns[k] = parseCustomFieldValue(raw)
	}
	if assignee != nil {
		issue.Fields.Unknowns["assignee"] = assignee
	}

	created, resp, err := c.Issue.Create(issue)
	if err != nil {
		return nil, wrapError(resp, err)
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
		return nil, wrapError(resp, err)
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
		return nil, wrapError(resp, err)
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
		return nil, nil, wrapError(resp, err)
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
		return nil, wrapError(resp, err)
	}
	return project.IssueTypes, nil
}

// TestConnection verifies the connection to Jira works
func (c *Client) TestConnection() error {
	_, resp, err := c.User.GetSelf()
	return wrapError(resp, err)
}

// GetEpics returns open epics in the given project
func (c *Client) GetEpics(projectKey string) ([]jira.Issue, error) {
	jql := fmt.Sprintf("project = %s AND issuetype = Epic AND resolution = Unresolved ORDER BY created DESC", projectKey)
	return c.SearchIssues(jql, 100)
}

var accountIDRE = regexp.MustCompile(`^([0-9a-f]{24}|\d+:[0-9a-f-]{36})$`)

// ResolveAccountID turns a configured assignee (email, name or account ID)
// into a Jira Cloud account ID. Exactly one active user must match.
func (c *Client) ResolveAccountID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if accountIDRE.MatchString(value) {
		return value, nil
	}

	c.accountMu.Lock()
	defer c.accountMu.Unlock()
	if id, ok := c.accountID[value]; ok {
		return id, nil
	}

	req, err := c.NewRequest("GET", "rest/api/3/user/search?query="+url.QueryEscape(value), nil)
	if err != nil {
		return "", err
	}
	var users []jira.User
	resp, err := c.Do(req, &users)
	if err != nil {
		err = wrapError(resp, err)
		if StatusOf(err) == 403 {
			return "", fmt.Errorf("cannot look up assignee %q (no permission to browse users); set issue_defaults.assignee to an account ID instead", value)
		}
		return "", fmt.Errorf("assignee lookup for %q failed: %w", value, err)
	}

	var active []jira.User
	for _, u := range users {
		if u.Active {
			active = append(active, u)
		}
	}
	// Prefer an exact email or display-name match when the search is fuzzy.
	if len(active) > 1 {
		for _, u := range active {
			if strings.EqualFold(u.EmailAddress, value) || strings.EqualFold(u.DisplayName, value) {
				active = []jira.User{u}
				break
			}
		}
	}

	switch len(active) {
	case 1:
		c.accountID[value] = active[0].AccountID
		return active[0].AccountID, nil
	case 0:
		return "", fmt.Errorf("issue_defaults.assignee %q matches no active Jira user", value)
	default:
		names := make([]string, 0, len(active))
		for _, u := range active {
			label := u.DisplayName
			if u.EmailAddress != "" {
				label += " <" + u.EmailAddress + ">"
			}
			names = append(names, label)
		}
		return "", fmt.Errorf("issue_defaults.assignee %q matches several users: %s", value, strings.Join(names, ", "))
	}
}
