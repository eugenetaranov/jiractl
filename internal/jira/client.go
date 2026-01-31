package jira

import (
	"fmt"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/e/jiractl/internal/config"
	"github.com/e/jiractl/internal/keyring"
)

type Client struct {
	*jira.Client
	config *config.Config
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

// CreateIssue creates a new issue in Jira
func (c *Client) CreateIssue(project, issueType, summary, description string) (*jira.Issue, error) {
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

	created, resp, err := c.Issue.Create(issue)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("failed to create issue (status %d): %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	return created, nil
}

// SearchIssues searches for issues using JQL
func (c *Client) SearchIssues(jql string, maxResults int) ([]jira.Issue, error) {
	if maxResults <= 0 {
		maxResults = 50
	}

	opts := &jira.SearchOptions{
		MaxResults: maxResults,
		Fields:     []string{"key", "summary", "status", "assignee", "priority", "created", "updated"},
	}

	issues, resp, err := c.Issue.Search(jql, opts)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("search failed (status %d): %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return issues, nil
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
