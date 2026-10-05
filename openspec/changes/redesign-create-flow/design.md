## Context

`runCreate` is a straight sequence of prompts with blocking network calls between them (`GetIssueTypes`, `GetEpics`, `GetIssue` for the epic summary). The fields to send are scattered between `create.go` and `client.CreateIssue`.

## Goals / Non-Goals

**Goals:** no visible wait between prompts; one place where the full payload is assembled and shown; a fully scriptable path.

**Non-Goals:** editing existing issues; per-field interactive pickers for arbitrary custom fields. `-F` and `e` take raw values.

## Decisions

### `issueDraft` as the single source of truth
Every input source fills one struct: `issueDraft{Type, Summary, Description, Epic, Assignee, Components, Labels, Fields map[string]any}`. The sources are config defaults, then the saved draft, then flags, then prompts, then edits. `client.CreateIssue(draft)` sends exactly what the review screen shows. This removes the hidden default handling from the client.

### Background prefetch
At the start, launch goroutines for issue types, epics, `GET /rest/api/2/field` (to map id → name), the default epic's summary, and the assignee lookup. Each one writes to a `prefetch` struct with a `sync.WaitGroup` per item. A prompt that needs a value waits only on that item. If the wait exceeds 300 ms, a spinner is shown on stderr. Errors are kept and reported only when the item is actually needed.
- Alternative: one up-front parallel fetch that blocks before the first prompt. Rejected, because it still makes the user wait before they can type anything.

### Description input
In the multiline prompt, a line with only `.` or Ctrl+D finishes input. Typing `:e` on its own line opens `$EDITOR` (falling back to `$VISUAL`, then `vi`) on a temp file pre-filled with what was typed so far. Blank lines are kept.

### Review and confirm loop
The review screen shows every field in the payload, with labels from the field-name map: `Story Points` instead of `customfield_10016`. Then `Create? [Y/e/d/n]`:
- `Y` or Enter creates the issue.
- `e` opens a picker of field names, then a prompt pre-filled with the current value. Type and epic reuse their pickers.
- `d` opens `$EDITOR` on the description.
- `n` saves a draft and cancels (exit 130).
After `e` or `d`, the review screen is shown again.

### Flags and non-interactive mode
If `-s` is given and either `-y` is set or stdin is not a TTY, no prompts run. Missing required values (type, when there is no default) are an error. `-d -` reads stdin to EOF. `-F key=val` accepts a field name or ID. Names are resolved through the field map, and the value is parsed with the existing `parseCustomFieldValue`. On success, stdout gets only the issue key, and the URL goes to stderr, so `KEY=$(jiractl create -s … -y)` works.

## Risks / Trade-offs

- [`/rest/api/2/field` is large on big instances] → Fetch it once per run, in the background. If it fails, show raw IDs.
- [The `.` terminator surprises users] → The prompt header says `(finish with "." on its own line, Ctrl+D, or ":e" for editor)`.
- [`$EDITOR` is unset on Windows] → Fall back to `notepad` on Windows.
