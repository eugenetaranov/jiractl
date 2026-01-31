package keyring

import (
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	ServiceName = "jiractl"
	UsernameKey = "username"
	TokenKey    = "token"
)

// GetUsername retrieves the username from the system keyring
func GetUsername() (string, error) {
	username, err := keyring.Get(ServiceName, UsernameKey)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", nil
		}
		return "", fmt.Errorf("failed to get username from keyring: %w", err)
	}
	return strings.TrimSpace(username), nil
}

// SetUsername stores the username in the system keyring
func SetUsername(username string) error {
	if err := keyring.Set(ServiceName, UsernameKey, strings.TrimSpace(username)); err != nil {
		return fmt.Errorf("failed to set username in keyring: %w", err)
	}
	return nil
}

// GetToken retrieves the API token from the system keyring
func GetToken() (string, error) {
	token, err := keyring.Get(ServiceName, TokenKey)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", nil
		}
		return "", fmt.Errorf("failed to get token from keyring: %w", err)
	}
	return strings.TrimSpace(token), nil
}

// SetToken stores the API token in the system keyring
func SetToken(token string) error {
	if err := keyring.Set(ServiceName, TokenKey, strings.TrimSpace(token)); err != nil {
		return fmt.Errorf("failed to set token in keyring: %w", err)
	}
	return nil
}

// GetCredentials retrieves both username and token
func GetCredentials() (username, token string, err error) {
	username, err = GetUsername()
	if err != nil {
		return "", "", err
	}
	token, err = GetToken()
	if err != nil {
		return "", "", err
	}
	return username, token, nil
}

// HasCredentials checks if both username and token are stored
func HasCredentials() bool {
	username, _ := GetUsername()
	token, _ := GetToken()
	return username != "" && token != ""
}

// ClearCredentials removes both username and token from keyring
func ClearCredentials() error {
	_ = keyring.Delete(ServiceName, UsernameKey)
	_ = keyring.Delete(ServiceName, TokenKey)
	return nil
}
