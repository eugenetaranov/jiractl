## ADDED Requirements

### Requirement: Deployment type detection
`configure` SHALL detect whether the server is Cloud or Server/Data Center and store the result. A config without it SHALL be treated as Cloud.

#### Scenario: Data Center detected
- **WHEN** `serverInfo` reports `deploymentType` other than `Cloud`
- **THEN** the config stores `deployment = "server"`, and the user is asked for a personal access token instead of an email and API token

### Requirement: Personal access token auth
For Server/DC, jiractl SHALL authenticate with `Authorization: Bearer <PAT>`.

#### Scenario: PAT login
- **WHEN** `deployment = "server"` and a valid PAT is stored
- **THEN** `auth test` succeeds without a username

### Requirement: Search works on Server/DC
For Server/DC, queries SHALL use the `/rest/api/2/search` endpoint.

#### Scenario: Saved query on Data Center
- **WHEN** the user runs `jiractl query mine` against a Data Center instance
- **THEN** the results are returned the same way as on Cloud

### Requirement: Epic Link field fallback
When the project does not accept `parent` for epics, jiractl SHALL set the Epic Link custom field instead. `issue_defaults.epic_field` SHALL override detection.

#### Scenario: Classic project
- **WHEN** the create screen has no `parent` but has a field of custom type `gh-epic-link`
- **THEN** the issue is created with that field set to the epic key

#### Scenario: No epic support
- **WHEN** neither `parent` nor an Epic Link field is available
- **THEN** a warning is printed to stderr, and the issue is created without an epic

### Requirement: Assignee by name on Server/DC
For Server/DC, the assignee default SHALL be sent as `assignee.name`, with no account ID lookup.

#### Scenario: DC assignee
- **WHEN** `deployment = "server"` and `assignee = "jdoe"`
- **THEN** the create request contains `"assignee": {"name": "jdoe"}`
