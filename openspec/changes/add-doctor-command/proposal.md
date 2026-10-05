## Why

When create or query fails, the user finds out one problem at a time, in the middle of a flow. Examples: a wrong project key, an expired token, a missing "Create issues" permission, a default epic that was closed, a custom field that isn't on the create screen, or a saved query with broken JQL. `jiractl doctor` checks everything up front and says exactly what to fix.

## What Changes

- New `jiractl doctor` command. It runs ordered checks and prints a ✓ / ! / ✗ line for each, with a fix hint for every warning or failure.
- Checks: config file (parse, permissions, unknown keys); credentials in the keyring; server reachable and deployment type; authentication; project exists; project permissions (browse, create, assign, transition, comment); default issue type, epic, component, assignee, and custom fields valid for the project; every saved query's JQL parses.
- Checks that depend on a failed check are shown as skipped.
- Exit 0 when nothing failed (warnings allowed), exit 1 on any failure. `--json` gives machine-readable output.
- `configure` ends by running doctor, so the user sees what to fix right away.

## Capabilities

### New Capabilities
- `doctor-command`: readiness checks for creating issues and running queries.

### Modified Capabilities
<!-- none -->

## Impact

- New `internal/doctor` package (check registry and runner), `internal/cmd/doctor.go`.
- Client additions: `MyPermissions(project, perms)`, `CreateMeta(project, type)`, `ParseJQL(queries)`, `ListComponents(project)`. Some of these are shared with the other changes (`ServerInfo`, user search).
- Works best after `fix-data-loss-and-silent-failures` (unknown-key detection, assignee resolution). Server/DC specifics come with `support-jira-server`.
