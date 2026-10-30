// Package keyring stores the Jira username and API token.
//
// Values are looked up in order: environment variables (JIRACTL_USERNAME,
// JIRACTL_TOKEN), the system keyring, then a 0600 credentials file under
// $XDG_CONFIG_HOME/jiractl. Writes go to the system keyring when it works
// and to the file otherwise (headless Linux has no Secret Service).
// JIRACTL_CREDENTIAL_STORE=keyring|file forces one backend.
package keyring

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/zalando/go-keyring"
)

const (
	ServiceName = "jiractl"
	UsernameKey = "username"
	TokenKey    = "token"

	UsernameEnv = "JIRACTL_USERNAME"
	TokenEnv    = "JIRACTL_TOKEN"
	StoreEnv    = "JIRACTL_CREDENTIAL_STORE"
)

// Source says where a credential value came from.
type Source string

const (
	SourceNone    Source = ""
	SourceEnv     Source = "env"
	SourceKeyring Source = "keyring"
	SourceFile    Source = "file"
)

var envNames = map[string]string{UsernameKey: UsernameEnv, TokenKey: TokenEnv}

// probe checks once per process whether the system keyring answers, so a
// missing D-Bus service costs one failed call, not one per lookup.
var (
	probeOnce  sync.Once
	keyringErr error
)

func probe() error {
	probeOnce.Do(func() {
		_, err := keyring.Get(ServiceName, UsernameKey)
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			keyringErr = err
		}
	})
	return keyringErr
}

// resetProbe is for tests that swap the keyring provider.
func resetProbe() {
	probeOnce = sync.Once{}
	keyringErr = nil
}

// forced returns the backend set by JIRACTL_CREDENTIAL_STORE, or "".
func forced() (Source, error) {
	switch v := strings.ToLower(strings.TrimSpace(os.Getenv(StoreEnv))); v {
	case "":
		return SourceNone, nil
	case "keyring":
		return SourceKeyring, nil
	case "file":
		return SourceFile, nil
	default:
		return SourceNone, fmt.Errorf("%s=%q: use keyring or file", StoreEnv, v)
	}
}

// KeyringError is why the system keyring can't be used, or nil when it
// can (or when the file store is forced and it is never asked).
func KeyringError() error {
	if f, _ := forced(); f == SourceFile {
		return nil
	}
	return probe()
}

// WriteStore is where SetUsername and SetToken save values.
func WriteStore() (Source, error) {
	f, err := forced()
	if err != nil {
		return SourceNone, err
	}
	if f != SourceNone {
		return f, nil
	}
	if probe() != nil {
		return SourceFile, nil
	}
	return SourceKeyring, nil
}

// WriteLocation describes WriteStore for messages: "system keyring" or the
// file path.
func WriteLocation() string {
	s, err := WriteStore()
	if err != nil || s == SourceKeyring {
		return "system keyring"
	}
	return DisplayPath()
}

func get(key string) (string, Source, error) {
	if v := strings.TrimSpace(os.Getenv(envNames[key])); v != "" {
		return v, SourceEnv, nil
	}
	f, err := forced()
	if err != nil {
		return "", SourceNone, err
	}
	if f != SourceFile {
		if err := probe(); err == nil {
			v, err := keyring.Get(ServiceName, key)
			switch {
			case err == nil && strings.TrimSpace(v) != "":
				return strings.TrimSpace(v), SourceKeyring, nil
			case err != nil && !errors.Is(err, keyring.ErrNotFound):
				return "", SourceNone, fmt.Errorf("failed to get %s from keyring: %w", key, err)
			}
		} else if f == SourceKeyring {
			return "", SourceNone, fmt.Errorf("system keyring unavailable: %w", err)
		}
	}
	if f == SourceKeyring {
		return "", SourceNone, nil
	}
	fc, err := readFile()
	if err != nil {
		return "", SourceNone, err
	}
	if v := fc.get(key); v != "" {
		return v, SourceFile, nil
	}
	return "", SourceNone, nil
}

func set(key, value string) error {
	value = strings.TrimSpace(value)
	store, err := WriteStore()
	if err != nil {
		return err
	}
	if store == SourceFile {
		return setFile(key, value)
	}
	if err := keyring.Set(ServiceName, key, value); err != nil {
		return fmt.Errorf("failed to set %s in keyring (set %s=file to use a file instead): %w", key, StoreEnv, err)
	}
	return nil
}

// GetUsername retrieves the username
func GetUsername() (string, error) {
	v, _, err := get(UsernameKey)
	return v, err
}

// SetUsername stores the username
func SetUsername(username string) error {
	return set(UsernameKey, username)
}

// GetToken retrieves the API token
func GetToken() (string, error) {
	v, _, err := get(TokenKey)
	return v, err
}

// SetToken stores the API token
func SetToken(token string) error {
	return set(TokenKey, token)
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

// Sources reports where the username and token currently come from.
func Sources() (username, token Source) {
	_, username, _ = get(UsernameKey)
	_, token, _ = get(TokenKey)
	return username, token
}

// HasCredentials checks if both username and token are stored
func HasCredentials() bool {
	username, _ := GetUsername()
	token, _ := GetToken()
	return username != "" && token != ""
}

// ClearCredentials removes username and token from the keyring and the
// credentials file. Environment variables are left alone.
func ClearCredentials() error {
	if KeyringError() == nil {
		_ = keyring.Delete(ServiceName, UsernameKey)
		_ = keyring.Delete(ServiceName, TokenKey)
	}
	return removeFile()
}
