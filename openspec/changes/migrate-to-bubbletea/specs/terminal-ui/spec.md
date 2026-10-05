## ADDED Requirements

### Requirement: Results stay on stdout only
Interactive screens SHALL render to stderr. Stdout SHALL carry only command results, so command output can be captured while prompts are shown.

#### Scenario: Capturing a key from an interactive create
- **WHEN** the user runs `KEY=$(jiractl create)` and completes the prompts
- **THEN** the prompts are visible on the terminal, and `KEY` contains only the new issue key

### Requirement: No interactive screen without a terminal
jiractl SHALL NOT start an interactive screen when stdin or stderr is not a terminal. It SHALL fall back to the non-interactive behavior or a "required: pass …" error.

#### Scenario: Piped stdin
- **WHEN** `echo body | jiractl create -s x -d -` runs
- **THEN** no interactive screen starts, and the issue is created from flags

### Requirement: Consistent keys
Every list SHALL support ↑/↓ and Ctrl+P/Ctrl+N to move, typing to filter, Enter to choose, Esc to go back or cancel, and Ctrl+C to cancel. Filtering SHALL treat space-separated words as terms that must all match.

#### Scenario: Multi-word filter
- **WHEN** the user types `change default` in the main menu
- **THEN** only "Change default epic" remains

#### Scenario: Ctrl+C anywhere
- **WHEN** the user presses Ctrl+C on any prompt or list outside the app shell
- **THEN** jiractl prints `Cancelled.` to stderr and exits 130

### Requirement: Multiline input
Multiline input (descriptions, comments) SHALL insert a newline on Enter, finish on Ctrl+D, open `$EDITOR` with the current text on Ctrl+E, and cancel on Esc. A line containing only `.` SHALL be kept as text.

#### Scenario: Paragraphs then Ctrl+D
- **WHEN** the user types `First`, Enter, Enter, `Second`, then presses Ctrl+D
- **THEN** the description is `First\n\nSecond`

#### Scenario: Dot is text
- **WHEN** the user types a line containing only `.` and then presses Ctrl+D
- **THEN** that `.` line is part of the text

### Requirement: Output stays visible
Prompts and pickers outside the app shell SHALL render inline with bounded height, and SHALL NOT clear output printed before them.

#### Scenario: Error before a picker
- **WHEN** a warning is printed and then a picker opens
- **THEN** the warning is still visible above the picker

### Requirement: Terminal restored
The terminal SHALL be restored (echo, cursor, screen) after normal exit, cancel, error or panic.

#### Scenario: Ctrl+C at secret prompt
- **WHEN** the user presses Ctrl+C at the API token prompt
- **THEN** typed characters echo again in the shell afterwards

### Requirement: Resizing and color
Screens SHALL re-layout on terminal resize, and SHALL use no color when `NO_COLOR` is set.

#### Scenario: Narrow terminal
- **WHEN** the query browser runs in a terminal narrower than 100 columns
- **THEN** the preview is shown below the list instead of beside it
