# cli-conventions Specification

## Purpose
TBD - created by archiving change fix-data-loss-and-silent-failures. Update Purpose after archive.
## Requirements
### Requirement: Errors print once
Each error SHALL be printed once to stderr, with no repeated prefix such as `failed to create issue: failed to create issue:`.

#### Scenario: Single error line
- **WHEN** any command returns an error
- **THEN** stderr contains the message once, and the exit status is 1

### Requirement: Jira error details are shown
When Jira returns `errorMessages` or field `errors`, jiractl SHALL show them as readable text and SHALL NOT show the raw JSON body.

#### Scenario: Create rejected by field validation
- **WHEN** Jira responds `{"errors":{"summary":"Summary is required"}}`
- **THEN** the error reads `summary: Summary is required`

#### Scenario: Bad JQL
- **WHEN** a query's JQL fails with Jira's explanation in `errorMessages`
- **THEN** the error includes that explanation

### Requirement: Uniform cancellation
Cancelling any prompt or picker with Ctrl+C, Ctrl+D, or Esc SHALL print `Cancelled.` to stderr and exit with status 130. The exceptions are a picker documented as optional, where Esc means "skip", and the main menu, where Esc exits with status 0.

#### Scenario: Ctrl+C at summary prompt
- **WHEN** the user presses Ctrl+C at the Summary prompt in `create`
- **THEN** stderr shows `Cancelled.`, and the exit status is 130

### Requirement: Diagnostics go to stderr
Warnings, progress messages, and the query banner (`Running query`, `JQL`) SHALL be written to stderr. Stdout SHALL carry only the command's result.

#### Scenario: Piping query output
- **WHEN** the user runs `jiractl query mine > out.txt`
- **THEN** `out.txt` does not contain `Running query` or `JQL:`

### Requirement: Width-aware text
Truncation SHALL never split a multi-byte character, and columns SHALL be aligned by terminal display width.

#### Scenario: Non-ASCII summary
- **WHEN** a summary has 70 characters, including Cyrillic or emoji, and is shown in a 60-column field
- **THEN** it is cut at a character boundary, ends in `…`, and the next column stays aligned

### Requirement: Version flag on every command
`--version` and `-v` SHALL print the version on the root command and on every subcommand.

#### Scenario: Subcommand version
- **WHEN** the user runs `jiractl query --version`
- **THEN** the version string is printed, and the exit status is 0

