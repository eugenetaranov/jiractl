## ADDED Requirements

### Requirement: Default epic shown in main menu
The main menu header SHALL show the configured default epic's key and summary, or `none` when no default is set.

#### Scenario: Default set
- **WHEN** `epic_link = "OPS-40"` and OPS-40's summary is "DevOps k8s cluster upgrade"
- **THEN** the menu header includes `Default epic: OPS-40 DevOps k8s cluster upgrade`

#### Scenario: No default
- **WHEN** `epic_link` is not set
- **THEN** the menu header includes `Default epic: none`

#### Scenario: Default unusable
- **WHEN** the default epic does not exist or is resolved
- **THEN** the header shows the key followed by `(not found)` or `(done)`

#### Scenario: Slow Jira
- **WHEN** the epic lookup takes longer than 1.5 seconds
- **THEN** the menu appears without waiting, with the key only

### Requirement: Epic always shown on create review
The create review screen SHALL always include an Epic line and SHALL mark an epic that came from config as `(default)`.

#### Scenario: Default applied
- **WHEN** the user creates an issue and the default epic OPS-40 applies
- **THEN** the review shows `Epic: OPS-40 DevOps k8s cluster upgrade (default)`

#### Scenario: No epic
- **WHEN** no epic applies
- **THEN** the review shows `Epic: (none)`

### Requirement: Change default epic from the menu
The main menu SHALL contain `Change default epic`. It SHALL open an epic picker listing the current default first, then `None: no default epic`, a row that opens the epic search, and the open epics. It SHALL save only `issue_defaults.epic_link`.

#### Scenario: Pick a new default
- **WHEN** the user picks `Change default epic`, chooses `Search all epics…`, enters `devops k8s`, and selects OPS-40
- **THEN** `epic_link` becomes `OPS-40`, the rest of `~/.jiractl.toml` is unchanged, and `Default epic set to OPS-40 …` is shown

#### Scenario: Clear the default
- **WHEN** the user selects `None: no default epic`
- **THEN** `epic_link` is removed from the config, and `create` asks for an epic again

#### Scenario: Esc
- **WHEN** the user presses Esc in the picker
- **THEN** the config is unchanged, and the menu is shown again
