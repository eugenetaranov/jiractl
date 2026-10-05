package jira

import (
	"io"
	"net/http"
	"strings"
)

// loginCheckTransport turns a silently failed login into a 401. Jira answers
// some requests with bad credentials (e.g. searches) as anonymous: 200 OK,
// empty results, and an X-Seraph-LoginReason header saying the login failed.
// Without this, an expired token looks like "no issues found".
type loginCheckTransport struct {
	next http.RoundTripper
}

func (t *loginCheckTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	next := t.next
	if next == nil {
		next = http.DefaultTransport
	}
	resp, err := next.RoundTrip(req)
	if err != nil || resp.StatusCode == http.StatusUnauthorized {
		return resp, err
	}
	reason := strings.ToUpper(resp.Header.Get("X-Seraph-LoginReason"))
	if strings.Contains(reason, "AUTHENTICATED_FAILED") || strings.Contains(reason, "AUTHENTICATION_DENIED") {
		_ = resp.Body.Close()
		resp.StatusCode = http.StatusUnauthorized
		resp.Status = "401 Unauthorized"
		resp.Body = io.NopCloser(strings.NewReader(""))
		resp.ContentLength = 0
	}
	return resp, nil
}
