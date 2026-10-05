package jira

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	jira "github.com/andygrunwald/go-jira"
)

// APIError is a non-2xx response from Jira with its messages parsed out of
// the `errorMessages` and `errors` fields.
type APIError struct {
	Status   int
	Messages []string
	Fields   map[string]string
}

func (e *APIError) Error() string {
	var parts []string
	parts = append(parts, e.Messages...)
	keys := make([]string, 0, len(e.Fields))
	for k := range e.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", k, e.Fields[k]))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d %s", e.Status, http.StatusText(e.Status))
	}
	return strings.Join(parts, "; ")
}

// HasField reports whether Jira rejected the given field.
func (e *APIError) HasField(name string) bool {
	_, ok := e.Fields[name]
	return ok
}

// StatusOf returns the HTTP status of an APIError, or 0 for other errors.
func StatusOf(err error) int {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Status
	}
	return 0
}

// wrapError turns a go-jira error into an APIError when there is a response
// to read; transport errors (DNS, TLS, refused connection) pass through.
func wrapError(resp *jira.Response, err error) error {
	if err == nil {
		return nil
	}
	if resp == nil || resp.Response == nil {
		return err
	}
	return newAPIError(resp.Response)
}

func newAPIError(resp *http.Response) *APIError {
	apiErr := &APIError{Status: resp.StatusCode}
	if resp.Body == nil {
		return apiErr
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var parsed struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
	}
	if json.Unmarshal(body, &parsed) == nil {
		apiErr.Messages = parsed.ErrorMessages
		apiErr.Fields = parsed.Errors
	}
	if len(apiErr.Messages) == 0 && len(apiErr.Fields) == 0 {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			apiErr.Messages = []string{"authentication failed (401): check username and API token"}
		case http.StatusForbidden:
			apiErr.Messages = []string{"permission denied (403)"}
		}
	}
	return apiErr
}
