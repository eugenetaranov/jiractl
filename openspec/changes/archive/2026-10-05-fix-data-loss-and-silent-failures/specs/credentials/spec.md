## ADDED Requirements

### Requirement: Terminal restored after secret prompt
When the user interrupts the hidden token prompt, jiractl SHALL restore terminal echo before exiting.

#### Scenario: Ctrl+C at token prompt
- **WHEN** the user presses Ctrl+C at `API Token:`
- **THEN** the shell shows typed characters again afterward, and the exit status is 130

### Requirement: Tokens are never partially displayed
`auth list` and `--debug` output SHALL NOT include any characters of the token. They MAY show its length. Short tokens SHALL NOT cause a crash.

#### Scenario: Short token
- **WHEN** the stored token has 3 characters and the user runs `auth list`
- **THEN** the output shows `Token: set (3 chars)`, and the command exits 0

#### Scenario: Debug output
- **WHEN** the user runs `auth test --debug`
- **THEN** the output shows the token length and no token characters
