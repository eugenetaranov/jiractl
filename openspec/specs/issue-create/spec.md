# issue-create Specification

## Purpose
TBD - created by archiving change redesign-create-flow. Update Purpose after archive.
## Requirements
### Requirement: Network data is prefetched
`create` SHALL start fetching issue types, epics, and field names when it starts. A prompt that does not need network data SHALL NOT wait for it.

#### Scenario: Summary prompt appears immediately
- **WHEN** the user runs `jiractl create` and a default issue type is configured
- **THEN** the Summary prompt appears before the epic and field requests finish

#### Scenario: Slow fetch
- **WHEN** a needed item has not arrived within 300 ms
- **THEN** a spinner with the item name is shown on stderr until it arrives or fails

### Requirement: Review shows the full payload with names
Before the create request is sent, the review screen SHALL list every field in the payload. Custom fields SHALL be labeled with their Jira display names.

#### Scenario: Custom field label
- **WHEN** `custom_fields` contains `customfield_10016 = "3"`, and that field is named "Story Points"
- **THEN** the review shows `Story Points: 3`

#### Scenario: Field names unavailable
- **WHEN** the field-name request fails
- **THEN** the review shows raw field IDs, and creation is still possible

### Requirement: Edit before create
The confirm prompt SHALL be `Create? [Y/e/d/n]`. `e` SHALL let the user change any listed field, `d` SHALL open the description in `$EDITOR`, and the review SHALL be shown again after each edit.

#### Scenario: Fix summary typo
- **WHEN** the user presses `e`, selects Summary, and enters a new value
- **THEN** the review is shown again with the new summary, and nothing has been sent to Jira

#### Scenario: Edit description in editor
- **WHEN** the user presses `d`
- **THEN** `$EDITOR` opens on the current description, and the saved file contents become the description

### Requirement: Multi-paragraph descriptions
Description input SHALL keep blank lines. It SHALL end on a line containing only `.` or on Ctrl+D. `:e` on its own line SHALL open `$EDITOR` with the text typed so far.

#### Scenario: Two paragraphs
- **WHEN** the user types `First`, an empty line, `Second`, then `.`
- **THEN** the description is `First\n\nSecond`

### Requirement: Scriptable create
`create` SHALL accept `-s/--summary`, `-t/--type`, `-e/--epic`, `-d/--description` (with `-` meaning stdin), repeatable `-F/--field key=val`, and `-y/--yes`. With `-y`, or when stdin is not a terminal, it SHALL NOT prompt. On success, it SHALL print only the issue key to stdout.

#### Scenario: Fully scripted
- **WHEN** the user runs `echo body | jiractl create -s "Fix login" -t Bug -d - -F "Story Points=3" -y`
- **THEN** an issue is created with that summary, type, description `body`, and Story Points 3, and stdout is exactly the key followed by a newline

#### Scenario: Missing required value in non-interactive mode
- **WHEN** `-y` is given, `-t` is omitted, and there is no default issue type
- **THEN** no issue is created, and the command exits 1 with `issue type required: pass -t or set issue_defaults.issue_type`

#### Scenario: Unknown field name
- **WHEN** `-F "Nonexistent=1"` matches no field name or ID
- **THEN** the command exits 1 and lists close matches

