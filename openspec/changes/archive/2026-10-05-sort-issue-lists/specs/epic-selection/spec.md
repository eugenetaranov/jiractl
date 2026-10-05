## ADDED Requirements

### Requirement: Epic lists ordered like issue lists
Every epic picker SHALL order epics by status category (In Progress, To Do, Done) and then by last update, newest first.

#### Scenario: Recently updated epic first
- **WHEN** two open epics were updated yesterday and a month ago
- **THEN** the one updated yesterday is listed first
