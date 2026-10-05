# credentials Specification

## Purpose
TBD - created by archiving change fix-data-loss-and-silent-failures. Update Purpose after archive.
## Requirements
### Requirement: Terminal restored after secret prompt
When the user interrupts the hidden token prompt, jiractl SHALL restore terminal echo before exiting.

#### Scenario: Ctrl+C at token prompt
- **WHEN** the user presses Ctrl+C at `API Token:`
- **THEN** the shell shows typed characters again afterward, and the exit status is 130

### Requirement: Tokens are never partially displayed
`auth list` and `--debug` output SHALL NOT include any characters of the token. They MAY show its length. Short tokens SHALL NOT cause a crash.

#### Scenario: Short token
- **WHEN** the stored token has 3 characters and the user runs `auth list`
- **THEN** the output shows `Token: set (3 chars)`, and the command exits 0

#### Scenario: Debug output
- **WHEN** the user runs `auth test --debug`
- **THEN** the output shows the token length and no token characters

### Requirement: Single credential flow
`configure` and `auth create` SHALL use the same credential prompt and validation. All "credentials missing" and "not configured" messages SHALL point to `jiractl configure`.

#### Scenario: auth create validates
- **WHEN** the user runs `jiractl auth create` and enters a token Jira rejects
- **THEN** nothing is saved, and the token is asked for again, the same way `configure` asks

#### Scenario: auth create without server
- **WHEN** `auth create` runs and no server is configured
- **THEN** the server is asked for and validated first

#### Scenario: Consistent hint
- **WHEN** any command fails because credentials are missing
- **THEN** the message says `run 'jiractl configure'`

