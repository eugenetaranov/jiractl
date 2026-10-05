## ADDED Requirements

### Requirement: Saves preserve unrelated content
Saving the config SHALL change only the keys whose values changed. It SHALL keep comments, key order, blank lines, and keys it does not manage.

#### Scenario: Comment survives a project change
- **WHEN** `~/.jiractl.toml` contains a comment line `# team board` and the user changes only `project`
- **THEN** the saved file still contains `# team board`, and only the `project` line differs

#### Scenario: Missing key is inserted under its table
- **WHEN** the file has an `[issue_defaults]` table without `issue_type`, and a save sets `issue_type`
- **THEN** `issue_type = "<value>"` is added inside `[issue_defaults]`, and no other line changes

### Requirement: Atomic, private writes
The config file SHALL be written to a temporary file in the same directory and renamed over the original. The result SHALL have mode `0600`.

#### Scenario: Write failure leaves original intact
- **WHEN** writing the temporary file fails partway
- **THEN** the original `~/.jiractl.toml` is unchanged, and the command reports the error

#### Scenario: Permissions
- **WHEN** the config is saved
- **THEN** the file mode is `0600`

### Requirement: Unknown keys are reported
When the config contains keys jiractl does not recognize, loading it SHALL print one warning per key to stderr, naming the key.

#### Scenario: Typo in a default
- **WHEN** the config contains `[issue_defaults]` with `asignee = "bob"`
- **THEN** stderr shows `warning: unknown config key "issue_defaults.asignee"`, and the command continues
