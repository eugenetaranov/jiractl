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

