## Why

Credentials live only in the OS keyring. On Linux that is Secret Service over D-Bus, which a VM or container without a desktop session does not have, so `configure` cannot save and every command fails. jiractl should work over SSH on a headless box.

## What Changes

- Credentials are read from `JIRACTL_USERNAME`/`JIRACTL_TOKEN`, then the system keyring, then `~/.config/jiractl/credentials` (`$XDG_CONFIG_HOME` respected).
- When the keyring is unavailable, `configure` and `auth create` save to the credentials file (mode 0600, atomic write). `JIRACTL_CREDENTIAL_STORE=keyring|file` forces a backend.
- `auth list`, `doctor` and the post-save messages say where credentials come from or were saved; "missing" messages no longer name the keyring.
- A credentials file readable by others produces a warning.
- "Copy key" uses OSC 52 over SSH or when no clipboard tool is installed.

## Capabilities

### Modified Capabilities
- `credentials`: where credentials are read from and saved to; copy over SSH.

## Impact

`internal/keyring` (new file backend, same public API plus `Sources`, `WriteLocation`, `KeyringError`, `FilePermError`), `internal/cmd/auth.go`, `internal/cmd/configure.go`, `internal/cmd/app.go`, `internal/doctor/checks.go`, README, e2e tests.
