## ADDED Requirements

### Requirement: Status bar
The app shell SHALL show a status bar with the default epic, the result of the last action, and the last error with its hint. Errors SHALL stay visible until the next action, without a "Press Enter" pause.

#### Scenario: Failed action
- **WHEN** a query fails because Jira rejects the token
- **THEN** the status bar shows the error and the hint to create a new token, and the menu stays usable

#### Scenario: Successful create
- **WHEN** an issue is created from the menu
- **THEN** the status bar shows `Created OPS-99` with its URL
