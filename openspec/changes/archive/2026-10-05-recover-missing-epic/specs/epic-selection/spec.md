## ADDED Requirements

### Requirement: Missing epic does not abort create
When the epic for a new issue does not exist or can't be used, interactive `create` SHALL keep the entered values and offer epic recovery. It SHALL NOT exit with an error.

#### Scenario: Default epic deleted
- **WHEN** `issue_defaults.epic_link = "OPS-12"`, OPS-12 no longer exists, and the user enters a summary and description
- **THEN** jiractl shows `Epic OPS-12 not found` and opens the epic search, with the summary and description kept

#### Scenario: Rejected by Jira after send
- **WHEN** the epic check passed but Jira rejects the create request because of the parent field
- **THEN** the epic search opens, and after a choice the request is sent again without re-entering other fields

### Requirement: Search epics by keywords
The epic search SHALL ask for search words and query Jira for epics in the project whose summary matches every word, or whose key matches the input. Results SHALL be shown in a picker with unresolved epics listed first, and the picker SHALL offer to search again.

#### Scenario: Keyword search
- **WHEN** the user enters `devops k8s` at the search prompt
- **THEN** the picker lists epics whose summaries contain words starting with `devops` and `k8s`, such as `OPS-40 DevOps: k8s cluster upgrade`

#### Scenario: Search by key
- **WHEN** the user enters `OPS-40` at the search prompt
- **THEN** OPS-40 is listed

### Requirement: Skip epic
The epic picker SHALL always offer `Skip: create without epic`. Choosing it, or pressing Esc, SHALL continue creating the issue with no epic.

#### Scenario: Skip
- **WHEN** the user selects `Skip: create without epic`
- **THEN** the review shows `Epic: (none)`, and confirming creates the issue without a parent

### Requirement: Offer to update the default
When the missing epic came from `issue_defaults.epic_link` and the user picks a different epic, jiractl SHALL ask whether to save it as the new default. The default answer SHALL be no.

#### Scenario: Save new default
- **WHEN** the user picks OPS-40 and answers `y` to `Save OPS-40 as default epic? [y/N]`
- **THEN** `epic_link` in `~/.jiractl.toml` becomes `OPS-40`, and the rest of the file is unchanged

### Requirement: Non-interactive handling
With `-y` or without a TTY, a missing epic SHALL cause exit 1 with up to 5 suggested epics, unless `--no-epic` is given. `--no-epic` SHALL create the issue without an epic.

#### Scenario: Scripted with missing epic
- **WHEN** `jiractl create -s "x" -e OPS-12 -y` runs and OPS-12 does not exist
- **THEN** nothing is created, the command exits 1, and stderr lists candidate epics

#### Scenario: Scripted skip
- **WHEN** `jiractl create -s "x" --no-epic -y` runs while a default epic is configured
- **THEN** the issue is created without an epic
