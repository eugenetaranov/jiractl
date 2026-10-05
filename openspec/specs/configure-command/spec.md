# configure-command Specification

## Purpose
TBD - created by archiving change fix-data-loss-and-silent-failures. Update Purpose after archive.
## Requirements
### Requirement: Validate before saving
`jiractl configure` SHALL test the server, credentials, and project with the values entered in memory. It SHALL write to the keyring or config file only after the tests pass.

#### Scenario: Typo in project does not overwrite working setup
- **WHEN** a configured user re-runs `configure` and enters a project key that does not exist
- **THEN** the existing config file and keyring entries are unchanged, and the user is asked for the project key again

#### Scenario: Bad token is re-asked
- **WHEN** the connection test returns 401 or 403
- **THEN** the user is asked for the API token again, and nothing has been saved

#### Scenario: Unreachable server is re-asked
- **WHEN** the server URL cannot be reached or is not a Jira instance
- **THEN** the user is asked for the server URL again

### Requirement: Failed configure exits non-zero
`configure` SHALL exit with status 1 when validation still fails after 3 attempts on a field. It SHALL NOT print "Configuration saved!" unless something was actually saved.

#### Scenario: Repeated failure
- **WHEN** the connection test fails 3 times in a row
- **THEN** the command prints the last error once and exits 1

### Requirement: Skipping an optional picker keeps the previous value
Pressing Esc in the default issue type or default epic picker SHALL keep the previously configured value. The summary SHALL list what was saved.

#### Scenario: Esc on epic picker
- **WHEN** the user presses Esc in "Select default epic"
- **THEN** the existing `epic_link` is kept, and the summary shows it unchanged

### Requirement: Per-step validation
`configure` SHALL validate each value as soon as it is entered, before asking for the next one.

#### Scenario: Missing scheme
- **WHEN** the user enters `acme.atlassian.net`
- **THEN** it is treated as `https://acme.atlassian.net` and checked right away

#### Scenario: Token checked before project prompt
- **WHEN** the user enters an invalid token
- **THEN** they are told it was rejected and asked for the token again before any project prompt

### Requirement: Project picker
`configure` SHALL offer the projects visible to the user in a picker, with the current project preselected. Typing a key directly SHALL still be possible.

#### Scenario: Pick project
- **WHEN** the credentials are valid
- **THEN** a picker lists `KEY - Name` for each accessible project

### Requirement: Starter queries
When the config has no queries, `configure` SHALL add the starter queries `mine`, `recent`, and `unassigned` in the same single save.

#### Scenario: First setup
- **WHEN** first-time setup completes
- **THEN** `~/.jiractl.toml` contains the three starter queries, and `jiractl query mine` works

#### Scenario: Existing queries untouched
- **WHEN** the config already has at least one query
- **THEN** no starter queries are added

