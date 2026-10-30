## ADDED Requirements

### Requirement: Credential lookup order
jiractl SHALL read each credential value from the `JIRACTL_USERNAME`/`JIRACTL_TOKEN` environment variables first, then the system keyring, then the credentials file at `$XDG_CONFIG_HOME/jiractl/credentials` (default `~/.config/jiractl/credentials`).

#### Scenario: Env token overrides stored token
- **WHEN** a token is stored and `JIRACTL_TOKEN` is set
- **THEN** jiractl uses the env token, and the stored username still applies

#### Scenario: Env only
- **WHEN** no keyring is available, no file exists, and both env vars are set
- **THEN** commands authenticate with the env values

### Requirement: File store when the keyring is unavailable
When the system keyring cannot be reached, `configure` and `auth create` SHALL save credentials to the credentials file with mode 0600. `JIRACTL_CREDENTIAL_STORE=keyring|file` SHALL force a backend.

#### Scenario: Headless Linux
- **WHEN** the user runs `jiractl configure` on Linux with no Secret Service
- **THEN** credentials are written to `~/.config/jiractl/credentials` with mode 0600, and the summary names that path

#### Scenario: Forced keyring unavailable
- **WHEN** `JIRACTL_CREDENTIAL_STORE=keyring` is set and the keyring cannot be reached
- **THEN** saving fails with an error that suggests `JIRACTL_CREDENTIAL_STORE=file`

### Requirement: Credentials file permissions
jiractl SHALL warn, and still read the file, when the credentials file is readable or writable by group or others.

#### Scenario: Mode 0644
- **WHEN** the credentials file has mode 0644
- **THEN** a warning with the `chmod 600` fix is printed to stderr, and `doctor` reports the credentials check as a warning

### Requirement: Credential source is shown
`auth list` and `doctor` SHALL say where each credential value comes from: environment, system keyring, or the file path.

#### Scenario: Mixed sources
- **WHEN** the username is in the file and `JIRACTL_TOKEN` is set
- **THEN** `doctor` shows `username from file ~/.config/jiractl/credentials, token from env`

### Requirement: Copy key over SSH
"Copy key" SHALL use the terminal clipboard (OSC 52) when running over SSH or when no system clipboard tool is available.

#### Scenario: SSH session
- **WHEN** `SSH_TTY` is set and the user picks "Copy key"
- **THEN** jiractl sends the key with OSC 52, and the status bar shows `Copied <key> (terminal clipboard)`
