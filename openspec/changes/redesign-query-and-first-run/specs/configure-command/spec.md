## ADDED Requirements

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
