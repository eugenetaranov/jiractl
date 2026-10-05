package jira

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// MyPermissions reports which of perms the user has in project.
func (c *Client) MyPermissions(project string, perms []string) (map[string]bool, error) {
	endpoint := fmt.Sprintf("rest/api/2/mypermissions?projectKey=%s&permissions=%s",
		url.QueryEscape(project), url.QueryEscape(strings.Join(perms, ",")))
	req, err := c.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Permissions map[string]struct {
			HavePermission bool `json:"havePermission"`
		} `json:"permissions"`
	}
	resp, err := c.Do(req, &out)
	if err != nil {
		return nil, wrapError(resp, err)
	}
	have := map[string]bool{}
	for k, v := range out.Permissions {
		have[k] = v.HavePermission
	}
	return have, nil
}

// CreateField is a field on the create screen of an issue type.
type CreateField struct {
	ID            string
	Name          string
	Required      bool
	HasDefault    bool
	AllowedValues []string // value or name of each option, when the field is a list
}

// CreateMetaFields returns the create-screen fields for an issue type in a
// project. Handles both the Cloud ("issueTypes"/"fields") and Data Center
// ("values") response shapes.
func (c *Client) CreateMetaFields(project, issueType string) ([]CreateField, error) {
	base := "rest/api/2/issue/createmeta/" + url.PathEscape(project) + "/issuetypes"

	var types struct {
		IssueTypes []struct{ ID, Name string } `json:"issueTypes"`
		Values     []struct{ ID, Name string } `json:"values"`
	}
	if err := c.getJSON(base+"?maxResults=200", &types); err != nil {
		return nil, err
	}
	typeID := ""
	for _, t := range append(types.IssueTypes, types.Values...) {
		if strings.EqualFold(t.Name, issueType) {
			typeID = t.ID
		}
	}
	if typeID == "" {
		return nil, fmt.Errorf("issue type %q is not available in project %s", issueType, project)
	}

	type rawField struct {
		FieldID       string            `json:"fieldId"`
		Key           string            `json:"key"`
		Name          string            `json:"name"`
		Required      bool              `json:"required"`
		HasDefault    bool              `json:"hasDefaultValue"`
		AllowedValues []json.RawMessage `json:"allowedValues"`
	}
	var fields struct {
		Fields []rawField `json:"fields"`
		Values []rawField `json:"values"`
	}
	if err := c.getJSON(base+"/"+url.PathEscape(typeID)+"?maxResults=200", &fields); err != nil {
		return nil, err
	}

	var out []CreateField
	for _, f := range append(fields.Fields, fields.Values...) {
		cf := CreateField{ID: f.FieldID, Name: f.Name, Required: f.Required, HasDefault: f.HasDefault}
		if cf.ID == "" {
			cf.ID = f.Key
		}
		for _, raw := range f.AllowedValues {
			var v struct{ Value, Name string }
			if json.Unmarshal(raw, &v) == nil {
				if v.Value != "" {
					cf.AllowedValues = append(cf.AllowedValues, v.Value)
				} else if v.Name != "" {
					cf.AllowedValues = append(cf.AllowedValues, v.Name)
				}
			}
		}
		out = append(out, cf)
	}
	return out, nil
}

// ListComponents returns the names of a project's components.
func (c *Client) ListComponents(project string) ([]string, error) {
	var comps []struct {
		Name string `json:"name"`
	}
	if err := c.getJSON("rest/api/2/project/"+url.PathEscape(project)+"/components", &comps); err != nil {
		return nil, err
	}
	names := make([]string, len(comps))
	for i, comp := range comps {
		names[i] = comp.Name
	}
	return names, nil
}

// ParseJQL validates queries and returns Jira's errors per query (an empty
// slice means the query is valid). Instances without the parse endpoint fall
// back to running each query with a single result.
func (c *Client) ParseJQL(queries []string) ([][]string, error) {
	req, err := c.NewRequest("POST", "rest/api/3/jql/parse?validation=strict", map[string][]string{"queries": queries})
	if err != nil {
		return nil, err
	}
	var out struct {
		Queries []struct {
			Errors []string `json:"errors"`
		} `json:"queries"`
	}
	resp, err := c.Do(req, &out)
	if err != nil {
		err = wrapError(resp, err)
		if StatusOf(err) != 404 {
			return nil, err
		}
		results := make([][]string, len(queries))
		for i, q := range queries {
			if _, err := c.SearchIssues(q, 1); err != nil {
				results[i] = []string{err.Error()}
			}
		}
		return results, nil
	}
	results := make([][]string, len(queries))
	for i := range queries {
		if i < len(out.Queries) {
			results[i] = out.Queries[i].Errors
		}
	}
	return results, nil
}

func (c *Client) getJSON(endpoint string, v interface{}) error {
	req, err := c.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req, v)
	return wrapError(resp, err)
}
