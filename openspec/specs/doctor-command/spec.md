# doctor-command Specification

## Purpose
TBD - created by archiving change add-doctor-command. Update Purpose after archive.
## Requirements
### Requirement: Readiness checks
`jiractl doctor` SHALL check the config, credentials, server, authentication, project, project permissions, configured defaults, and saved queries. It SHALL print one result line per check.

#### Scenario: Everything healthy
- **WHEN** the setup is valid and the user can create issues in the project
- **THEN** every check shows OK, and the command exits 0

#### Scenario: Expired token
- **WHEN** the stored token is rejected with 401
- **THEN** the Authentication check fails with a hint to run `jiractl configure`, and the project, permission, default, and query checks are shown as skipped

### Requirement: Severity and exit status
Problems that would make create or query fail SHALL be reported as failures. Problems that only disable one action SHALL be warnings. The exit status SHALL be 1 if any check failed, otherwise 0.

#### Scenario: Missing transition permission only
- **WHEN** the user lacks TRANSITION_ISSUES but everything else passes
- **THEN** that check is a warning, and the command exits 0

#### Scenario: Missing create permission
- **WHEN** the user lacks CREATE_ISSUES in the configured project
- **THEN** that check fails with a hint naming the project, and the command exits 1

### Requirement: Defaults validated against the project
Doctor SHALL verify that the default issue type, component, epic, assignee, and custom fields are valid for the configured project and issue type.

#### Scenario: Custom field not on create screen
- **WHEN** `custom_fields` contains a field that is not on the create screen for the default issue type
- **THEN** the default fields check fails and names the field by display name and ID

#### Scenario: Closed default epic
- **WHEN** `epic_link` points to a resolved epic
- **THEN** the default epic check shows a warning

### Requirement: Saved queries validated
Doctor SHALL check each saved query's JQL, after `${project}` is expanded, and report each invalid query with Jira's error.

#### Scenario: Broken JQL
- **WHEN** query `recent` has `updated >= -7x`
- **THEN** the check fails for `recent` and shows Jira's parse error

### Requirement: Fix hints
Every warning and failure SHALL include a hint that names the command to run or the config key to change.

#### Scenario: Unknown component
- **WHEN** `component = "Backnd"` does not exist and `Backend` does
- **THEN** the hint says to set `issue_defaults.component` and suggests `Backend`

### Requirement: Machine-readable output
`doctor --json` SHALL print a JSON array of `{id, title, status, detail, hint}` and SHALL use the same exit status rules.

#### Scenario: JSON
- **WHEN** the user runs `jiractl doctor --json`
- **THEN** stdout is valid JSON, and `status` is one of `ok`, `warn`, `fail`, `skip`

