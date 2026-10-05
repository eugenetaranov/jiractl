## Why

jiractl loses or overwrites user data in several places and fails silently in others: `configure` saves before it tests the connection and exits 0 on failure, every save rewrites `~/.jiractl.toml` from scratch, a declined or rejected `create` throws away everything typed, and two config defaults (`component`, `assignee`) are never applied correctly. These are small, low-risk fixes that should ship before the larger create/query redesigns.

## What Changes

- `configure` tests the connection with in-memory values, saves only on success, re-asks the failing field, and exits non-zero on failure.
- Config saves change only the keys that changed, write atomically (temp file + rename), and use `0600` permissions.
- `create` confirm defaults to Yes (`[Y/n]`); on Jira rejection the entered values are saved as a draft and offered on the next run.
- `component` default is sent as `components`; `assignee` is resolved to a Jira Cloud account ID; unknown config keys produce a warning.
- Errors print once (`SilenceErrors`).
- Ctrl+C at the hidden token prompt restores the terminal.
- `auth list` no longer crashes on short tokens; `--debug` prints only the token length.
- One cancel handler: prints `Cancelled.` to stderr and exits 130 everywhere.
- Warnings, progress, and the "Running query / JQL" banner go to stderr.
- Summaries truncate by rune and columns align by display width.
- `--version` / `-v` works on every command via cobra's built-in version flag.
- Create errors show Jira's messages once, without raw JSON or a repeated prefix; JQL errors include Jira's explanation.
- README documents `inspect` and correctly describes when `create` asks for the issue type.

## Capabilities

### New Capabilities
- `config-file`: how `~/.jiractl.toml` is read, validated, and written.
- `configure-command`: validate-before-save behavior of `jiractl configure`.
- `issue-defaults`: how config defaults are applied to new issues, and how the create confirm step and drafts behave.
- `cli-conventions`: shared behavior for errors, cancellation, output streams, text width, and the version flag.
- `credentials`: safe token prompting and display.

### Modified Capabilities
<!-- none: no existing specs -->

## Impact

- Code: `internal/config/config.go`, `internal/cmd/{root,configure,create,auth,query}.go`, `internal/jira/client.go`, `README.md`.
- New dependency: `github.com/mattn/go-runewidth` (already an indirect dependency via go-fuzzyfinder).
- Exit codes change: cancel now exits 130, and a failed `configure` exits 1. Scripts that relied on exit 0 will notice.
