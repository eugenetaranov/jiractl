package keyring

import (
	"errors"
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

// setup isolates a test: empty env, temp config dir, and a mock keyring
// that either works (kerr == nil) or fails every call like headless Linux.
func setup(t *testing.T, kerr error) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(UsernameEnv, "")
	t.Setenv(TokenEnv, "")
	t.Setenv(StoreEnv, "")
	if kerr == nil {
		keyring.MockInit()
	} else {
		keyring.MockInitWithError(kerr)
	}
	resetProbe()
	t.Cleanup(resetProbe)
}

var errNoDBus = errors.New("org.freedesktop.secrets was not provided")

func TestKeyringAvailable(t *testing.T) {
	setup(t, nil)
	if err := SetUsername("me@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := SetToken(" tok "); err != nil {
		t.Fatal(err)
	}
	u, tok, err := GetCredentials()
	if err != nil || u != "me@example.com" || tok != "tok" {
		t.Fatalf("got %q %q %v", u, tok, err)
	}
	if su, st := Sources(); su != SourceKeyring || st != SourceKeyring {
		t.Fatalf("sources %q %q", su, st)
	}
	p, _ := FilePath()
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("credentials file should not exist: %v", err)
	}
}

func TestFallbackToFile(t *testing.T) {
	setup(t, errNoDBus)
	if s := must(WriteStore()); s != SourceFile {
		t.Fatalf("write store %q", s)
	}
	if err := SetUsername("me@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := SetToken("tok"); err != nil {
		t.Fatal(err)
	}
	// Writing the username again keeps the token.
	if err := SetUsername("you@example.com"); err != nil {
		t.Fatal(err)
	}
	u, tok, err := GetCredentials()
	if err != nil || u != "you@example.com" || tok != "tok" {
		t.Fatalf("got %q %q %v", u, tok, err)
	}
	p, _ := FilePath()
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %04o", info.Mode().Perm())
	}
	if KeyringError() == nil {
		t.Fatal("KeyringError should report the keyring failure")
	}
	if err := ClearCredentials(); err != nil {
		t.Fatal(err)
	}
	if HasCredentials() {
		t.Fatal("credentials left after clear")
	}
}

func TestEnvOverrides(t *testing.T) {
	setup(t, nil)
	_ = SetUsername("stored@example.com")
	_ = SetToken("stored")
	t.Setenv(TokenEnv, "fromenv")
	u, tok, _ := GetCredentials()
	if u != "stored@example.com" || tok != "fromenv" {
		t.Fatalf("got %q %q", u, tok)
	}
	if su, st := Sources(); su != SourceKeyring || st != SourceEnv {
		t.Fatalf("sources %q %q", su, st)
	}
}

func TestEnvWithoutAnyStore(t *testing.T) {
	setup(t, errNoDBus)
	t.Setenv(UsernameEnv, "me@example.com")
	t.Setenv(TokenEnv, "tok")
	if !HasCredentials() {
		t.Fatal("env credentials not found")
	}
}

func TestForcedFile(t *testing.T) {
	setup(t, nil)
	t.Setenv(StoreEnv, "file")
	if err := SetToken("tok"); err != nil {
		t.Fatal(err)
	}
	if v, _ := keyring.Get(ServiceName, TokenKey); v != "" {
		t.Fatalf("keyring was written: %q", v)
	}
	if _, st := Sources(); st != SourceFile {
		t.Fatalf("source %q", st)
	}
}

func TestForcedKeyringUnavailable(t *testing.T) {
	setup(t, errNoDBus)
	t.Setenv(StoreEnv, "keyring")
	if err := SetToken("tok"); err == nil {
		t.Fatal("expected an error when the forced keyring fails")
	}
}

func TestBadStoreValue(t *testing.T) {
	setup(t, nil)
	t.Setenv(StoreEnv, "vault")
	if _, err := GetToken(); err == nil {
		t.Fatal("expected an error for an unknown store")
	}
}

func TestLoosePermissions(t *testing.T) {
	setup(t, errNoDBus)
	_ = SetToken("tok")
	p, _ := FilePath()
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if FilePermError() == nil {
		t.Fatal("expected a permissions error")
	}
	if tok, _ := GetToken(); tok != "tok" {
		t.Fatalf("token %q", tok)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
