## ADDED Requirements

### Requirement: Results ordered by status category and recency
Interactive results and the table output SHALL list In Progress issues first, then To Do, then Done, and within each group the most recently updated first. `-o keys` and `-o json` SHALL keep Jira's order.

#### Scenario: Done issues last
- **WHEN** a query returns a Done issue updated an hour ago and a To Do issue updated last week
- **THEN** the To Do issue is listed before the Done issue

#### Scenario: Recent first within a group
- **WHEN** two To Do issues were updated 2 days and 5 minutes ago
- **THEN** the one updated 5 minutes ago is listed first

#### Scenario: Custom done status
- **WHEN** an issue's status is "Completed" in Jira's done category
- **THEN** it is listed with the Done issues

#### Scenario: Scripts keep JQL order
- **WHEN** the user runs `jiractl query mine -o keys`
- **THEN** keys are printed in the order Jira returned them

### Requirement: Updated column
Result rows SHALL show how long ago each issue was updated, as a short age such as `5m`, `3h`, `2d`, `3w` or `4mo`.

#### Scenario: Age shown
- **WHEN** an issue was updated 3 hours ago
- **THEN** its row shows `3h`
