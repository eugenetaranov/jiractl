//go:build keyringmock

package keyring

import (
	"os"

	"github.com/zalando/go-keyring"
)

// Test builds (-tags keyringmock) use an in-memory keyring seeded from
// JIRACTL_TEST_USERNAME / JIRACTL_TEST_TOKEN, so end-to-end tests never touch
// the real system keyring.
func init() {
	keyring.MockInit()
	if u := os.Getenv("JIRACTL_TEST_USERNAME"); u != "" {
		_ = keyring.Set(ServiceName, UsernameKey, u)
	}
	if t := os.Getenv("JIRACTL_TEST_TOKEN"); t != "" {
		_ = keyring.Set(ServiceName, TokenKey, t)
	}
}
