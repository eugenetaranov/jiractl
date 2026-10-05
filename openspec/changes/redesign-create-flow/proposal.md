## Why

Creating an issue is the main job of jiractl, and the flow is slow and inflexible. It stalls between steps waiting on the network. It shows fields as `customfield_XXXXX`. The only choices at the end are create or cancel. Descriptions can't contain paragraphs. It can't be scripted.

## What Changes

- Fetch issue types, epics, field names, and the assignee lookup in the background as soon as `create` starts.
- The review screen lists every field that will be sent, with human-readable names.
- Confirm becomes `Create? [Y/e/d/n]`: `e` edits one field, `d` opens the description in `$EDITOR`.
- Description input supports blank lines: it ends on a line containing only `.` or on Ctrl+D, or the user opens `$EDITOR`.
- Non-interactive flags: `-s/--summary`, `-t/--type`, `-e/--epic`, `-d/--description` (`-` reads stdin), `-F/--field key=val` (repeatable), `-y/--yes`.
- **BREAKING**: an empty line no longer ends description input.

Depends on `fix-data-loss-and-silent-failures`, which provides drafts, `ErrCancelled`, `APIError`, and the assignee resolution. Archive that change first.

## Capabilities

### New Capabilities
- `issue-create`: the interactive and scripted flow of `jiractl create`.

### Modified Capabilities
- `issue-defaults`: the confirm prompt changes from `[Y/n]` to `[Y/e/d/n]`.

## Impact

- Code: `internal/cmd/create.go` (rewritten around an `issueDraft` struct), `internal/cmd/root.go` (prompt helpers), `internal/jira/client.go` (new `GetFieldNames` from `/rest/api/2/field`), `README.md`.
- No new dependencies. `$EDITOR` is run through `os/exec`.
