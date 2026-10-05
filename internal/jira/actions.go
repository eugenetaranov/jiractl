package jira

import (
	"fmt"
	"net/url"

	jira "github.com/andygrunwald/go-jira"
)

// ServerInfo is the subset of /rest/api/2/serverInfo jiractl uses.
type ServerInfo struct {
	BaseURL        string `json:"baseUrl"`
	Version        string `json:"version"`
	DeploymentType string `json:"deploymentType"`
	ServerTitle    string `json:"serverTitle"`
}

// GetServerInfo works without authentication, so it can check a URL before
// credentials are entered.
func (c *Client) GetServerInfo() (*ServerInfo, error) {
	req, err := c.NewRequest("GET", "rest/api/2/serverInfo", nil)
	if err != nil {
		return nil, err
	}
	var info ServerInfo
	resp, err := c.Do(req, &info)
	if err != nil {
		return nil, wrapError(resp, err)
	}
	if info.Version == "" && info.DeploymentType == "" {
		return nil, fmt.Errorf("not a Jira server (no version in serverInfo)")
	}
	return &info, nil
}

// Myself returns the authenticated user, cached for the process.
func (c *Client) Myself() (*jira.User, error) {
	c.selfMu.Lock()
	defer c.selfMu.Unlock()
	if c.self != nil {
		return c.self, nil
	}
	u, resp, err := c.User.GetSelf()
	if err != nil {
		return nil, wrapError(resp, err)
	}
	c.self = u
	return u, nil
}

// Project is a project visible to the user.
type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// ListProjects returns every project the user can see.
func (c *Client) ListProjects() ([]Project, error) {
	req, err := c.NewRequest("GET", "rest/api/2/project", nil)
	if err != nil {
		return nil, err
	}
	var projects []Project
	resp, err := c.Do(req, &projects)
	if err != nil {
		return nil, wrapError(resp, err)
	}
	return projects, nil
}

// GetTransitions lists the transitions available for an issue.
func (c *Client) GetTransitions(key string) ([]jira.Transition, error) {
	t, resp, err := c.Issue.GetTransitions(key)
	if err != nil {
		return nil, wrapError(resp, err)
	}
	return t, nil
}

// DoTransition moves an issue through a transition.
func (c *Client) DoTransition(key, transitionID string) error {
	resp, err := c.Issue.DoTransition(key, transitionID)
	return wrapError(resp, err)
}

// AssignToMe assigns the issue to the authenticated user.
func (c *Client) AssignToMe(key string) error {
	me, err := c.Myself()
	if err != nil {
		return err
	}
	body := map[string]string{"accountId": me.AccountID}
	if me.AccountID == "" {
		body = map[string]string{"name": me.Name} // Server/Data Center
	}
	req, err := c.NewRequest("PUT", "rest/api/2/issue/"+url.PathEscape(key)+"/assignee", body)
	if err != nil {
		return err
	}
	resp, err := c.Do(req, nil)
	return wrapError(resp, err)
}

// AddComment adds a plain-text comment to an issue.
func (c *Client) AddComment(key, text string) error {
	req, err := c.NewRequest("POST", "rest/api/2/issue/"+url.PathEscape(key)+"/comment", map[string]string{"body": text})
	if err != nil {
		return err
	}
	resp, err := c.Do(req, nil)
	return wrapError(resp, err)
}
