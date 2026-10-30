## 1. Credential store

- [x] 1.1 Probe the keyring once; add env lookup and `JIRACTL_CREDENTIAL_STORE`
- [x] 1.2 File backend: XDG path, atomic 0600 write keeping the other value, permission warning, delete
- [x] 1.3 `Sources`, `WriteLocation`, `KeyringError`, `FilePermError` helpers
- [x] 1.4 Unit tests with `MockInitWithError` for an unavailable keyring

## 2. Commands

- [x] 2.1 `auth list` shows the source; `auth delete` notes env vars it cannot remove; save messages name the location
- [x] 2.2 `doctor` credentials check: neutral "missing" messages, source in the OK detail, permissions warning
- [x] 2.3 "Copy key" falls back to OSC 52 over SSH or without a clipboard tool

## 3. Docs and tests

- [x] 3.1 README: credential order, headless Linux / SSH section
- [x] 3.2 e2e: `configure` with the file store; pin `XDG_CONFIG_HOME` inside the throwaway HOME
