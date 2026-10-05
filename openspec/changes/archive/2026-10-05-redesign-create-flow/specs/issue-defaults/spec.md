## MODIFIED Requirements

### Requirement: Confirm defaults to yes
The create confirmation SHALL be shown as `Create? [Y/e/d/n]`. Pressing Enter SHALL create the issue. `e` and `d` SHALL edit before creating, as described in `issue-create`.

#### Scenario: Enter confirms
- **WHEN** the user presses Enter at `Create? [Y/e/d/n]`
- **THEN** the issue is created

#### Scenario: Explicit no
- **WHEN** the user answers `n`
- **THEN** no issue is created, the entered values are saved as a draft, and the command exits 130
