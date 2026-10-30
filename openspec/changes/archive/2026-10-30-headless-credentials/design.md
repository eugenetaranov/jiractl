## Context

`internal/keyring` wraps `zalando/go-keyring`. On Linux without Secret Service every call fails with a D-Bus error rather than `ErrNotFound`.

## Decisions

- **Keep the package API.** Callers keep `GetCredentials`, `SetToken` and friends; backend choice is internal.
- **Probe once.** One `keyring.Get` per process decides whether the keyring works (nil or `ErrNotFound` means yes), so a missing D-Bus costs one failed call.
- **Per-value lookup.** Env, then keyring, then file, per key, so `JIRACTL_TOKEN` can override just the token.
- **File format.** TOML (`username`, `token`) via BurntSushi/toml, already a dependency. Written through temp file + rename, mode 0600, directory 0700. Writing one value keeps the other.
- **Loose permissions warn, not fail**, like ssh does for keys it can still read, so a chmod mistake doesn't lock the user out.
- **No permanent doctor warning on headless.** The file store is the intended mode there; doctor reports the source as OK and warns only on permissions.
- **OSC 52 first over SSH.** `xclip` on the remote host would fill the remote X clipboard, not the user's. Bubble Tea v2's `tea.SetClipboard` emits the sequence.

## Risks

- A running but locked keyring with no prompter can block in go-keyring. Documented workaround: `JIRACTL_CREDENTIAL_STORE=file` or env vars.
- The file is plaintext protected only by permissions; env vars let users keep the token in their own secret manager.
