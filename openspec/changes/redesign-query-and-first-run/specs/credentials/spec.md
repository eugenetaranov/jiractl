## ADDED Requirements

### Requirement: Single credential flow
`configure` and `auth create` SHALL use the same credential prompt and validation. All "credentials missing" and "not configured" messages SHALL point to `jiractl configure`.

#### Scenario: auth create validates
- **WHEN** the user runs `jiractl auth create` and enters a token Jira rejects
- **THEN** nothing is saved, and the token is asked for again, the same way `configure` asks

#### Scenario: auth create without server
- **WHEN** `auth create` runs and no server is configured
- **THEN** the server is asked for and validated first

#### Scenario: Consistent hint
- **WHEN** any command fails because credentials are missing
- **THEN** the message says `run 'jiractl configure'`
