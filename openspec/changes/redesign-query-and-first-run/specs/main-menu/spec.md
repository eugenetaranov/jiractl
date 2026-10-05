## ADDED Requirements

### Requirement: First-run detection
When no server or project is configured, an interactive jiractl invocation SHALL offer to run setup. After setup succeeds, it SHALL continue with what the user asked for.

#### Scenario: Bare jiractl before setup
- **WHEN** a new user runs `jiractl`
- **THEN** they are asked `jiractl isn't set up yet. Run setup now? [Y/n]`, and after setup they see the main menu

#### Scenario: Non-interactive before setup
- **WHEN** `jiractl query mine` runs with stdout not a terminal and no config exists
- **THEN** it exits 1 with `not configured: run 'jiractl configure'`

### Requirement: Menu loops
The main menu SHALL be shown again after each action finishes, is cancelled, or fails. It SHALL exit only on Exit or Esc.

#### Scenario: Return after create
- **WHEN** the user creates an issue from the menu
- **THEN** the created key is shown, and the menu appears again
