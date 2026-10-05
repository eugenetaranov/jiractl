# issue-defaults Specification

## Purpose
TBD - created by archiving change fix-data-loss-and-silent-failures. Update Purpose after archive.
## Requirements
### Requirement: Component default is sent
When `issue_defaults.component` is set, a created issue SHALL include that component in its `components` field.

#### Scenario: Component applied
- **WHEN** `component = "Backend"` and the user creates an issue
- **THEN** the create request contains `"components": [{"name": "Backend"}]`

### Requirement: Assignee default is resolved to an account ID
When `issue_defaults.assignee` is not already an account ID, jiractl SHALL resolve it through Jira user search and send `assignee.accountId`. It SHALL NOT send `assignee.name`.

#### Scenario: Unique match
- **WHEN** `assignee = "john.doe@example.com"` matches exactly one active user
- **THEN** the create request contains that user's `accountId`

#### Scenario: Ambiguous or no match
- **WHEN** the assignee matches zero users or more than one
- **THEN** no issue is created, and the error names the configured value and lists any candidates

### Requirement: Confirm defaults to yes
The create confirmation SHALL be shown as `Create? [Y/e/d/n]`. Pressing Enter SHALL create the issue. `e` and `d` SHALL edit before creating, as described in `issue-create`.

#### Scenario: Enter confirms
- **WHEN** the user presses Enter at `Create? [Y/e/d/n]`
- **THEN** the issue is created

#### Scenario: Explicit no
- **WHEN** the user answers `n`
- **THEN** no issue is created, the entered values are saved as a draft, and the command exits 130

### Requirement: Draft saved when create fails
When Jira rejects a create request, jiractl SHALL save the entered fields to a draft file with mode `0600` and print its path. The next `create` SHALL offer to resume the draft.

#### Scenario: Rejected create
- **WHEN** Jira returns 400 for the create request
- **THEN** the draft is written, its path is printed to stderr, and the command exits 1

#### Scenario: Resume
- **WHEN** a draft exists and the user runs `create` and accepts the resume prompt
- **THEN** summary, description, type, and epic are pre-filled from the draft, and the draft is deleted after a successful create

