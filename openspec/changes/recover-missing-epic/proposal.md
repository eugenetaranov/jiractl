## Why

When the configured or entered epic can't be found (deleted, moved, typo, no access), `create` sends it anyway and Jira rejects the request. The command exits with an error and everything typed is lost. The user should be able to pick a different epic, or skip it, and still create the issue.

## What Changes

- Before sending, check that the epic exists in Jira. This applies to every source of the epic: `issue_defaults.epic_link`, a resumed draft, or `-e`.
- If the epic is missing, show `Epic <KEY> not found` and open an epic search. The user types words such as `devops k8s`; matching epics are fetched from Jira as they type and shown in a picker.
- The picker always ends with `Skip: create without epic`.
- If Jira rejects the create request because of the parent or epic field anyway (for example, the epic exists but can't be a parent), offer the same recovery instead of exiting.
- Non-interactive mode (`-y` or no TTY): exit 1 with up to 5 suggested epics. `--no-epic` creates the issue without an epic.
- The normal epic picker (when there is no default) gets the same search-in-Jira behavior, so it is no longer limited to the 100 newest epics.

## Capabilities

### New Capabilities
- `epic-selection`: validating, searching, and skipping the epic during create.

### Modified Capabilities
<!-- none -->

## Impact

- Code: `internal/cmd/create.go`, `internal/jira/client.go` (`SearchEpics(project, text)`, and a check that tells whether a failed create was caused by the parent or epic field).
- Independent of the other changes; small enough to ship on its own or alongside `fix-data-loss-and-silent-failures`. `redesign-create-flow` should reuse the same picker for `e` → Epic.
