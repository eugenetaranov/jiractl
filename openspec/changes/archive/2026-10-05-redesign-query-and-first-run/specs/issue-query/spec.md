## ADDED Requirements

### Requirement: Results with preview
Interactive query results SHALL show a preview pane next to the list. The pane SHALL show the highlighted issue's type, status, priority, assignee, reporter, labels, and description.

#### Scenario: Moving the cursor updates preview
- **WHEN** the user moves the cursor to a different issue
- **THEN** the preview shows that issue's details without a full-screen redraw or a new command

### Requirement: Actions on an issue
Pressing Enter on an issue SHALL open an actions menu with: Open in browser, Copy key, Transition, Assign to me, Comment, Back.

#### Scenario: Transition
- **WHEN** the user picks Transition and selects "In Progress"
- **THEN** the issue is transitioned, its row shows the new status, and the actions menu is shown again

#### Scenario: Copy key without clipboard
- **WHEN** no clipboard is available and the user picks Copy key
- **THEN** the key is printed with a note that the clipboard is unavailable

### Requirement: Browse loop
After an action or Back, the user SHALL return to the results list. Esc in the results list SHALL leave the query with exit status 0, or return to the main menu when the query was started from it.

#### Scenario: Esc in results
- **WHEN** the user presses Esc in the results list after running `jiractl query mine`
- **THEN** the command exits 0 without printing `Cancelled.`

### Requirement: Ad-hoc JQL
`query --jql "<jql>"` SHALL run the given JQL without a saved query. `${project}` SHALL be expanded.

#### Scenario: One-off search
- **WHEN** the user runs `jiractl query --jql "project = ${project} AND labels = urgent"`
- **THEN** the results for that JQL are shown

### Requirement: Scriptable output
`-o keys` SHALL print one issue key per line. `-o json` SHALL print a JSON array. When stdout is not a terminal, no picker SHALL be shown.

#### Scenario: Piped output
- **WHEN** the user runs `jiractl query mine | wc -l`
- **THEN** no picker appears, and each issue is printed on its own line

#### Scenario: JSON output
- **WHEN** the user runs `jiractl query mine -o json`
- **THEN** stdout is a JSON array whose objects have `key`, `summary`, `status`, and `assignee`

### Requirement: Forgiving query names
Query names SHALL match exactly, then case-insensitively, then by unique prefix. An unmatched or ambiguous name SHALL produce an error that lists the candidates.

#### Scenario: Prefix
- **WHEN** queries `mine` and `unassigned` exist and the user runs `jiractl query MI`
- **THEN** `mine` runs

#### Scenario: No match
- **WHEN** the user runs `jiractl query foo`
- **THEN** the command exits 1 with `no query "foo"; available: mine, recent, unassigned`
