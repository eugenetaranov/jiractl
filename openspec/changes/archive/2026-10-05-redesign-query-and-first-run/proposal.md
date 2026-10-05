## Why

Both the query flow and the first-run experience end in dead ends. A query prints one issue and exits, with nothing you can do to it. A first-time user gets "not configured" errors, has no saved queries, and can't tell whether their project key was right. `configure` and `auth create` also set the same credentials with different rules.

## What Changes

- Query results open in fzf with the issue details in a preview pane.
- Enter on an issue opens an actions menu: open in browser, copy key, transition, assign to me, comment, back. The list stays open until Esc.
- `query --jql "<jql>"` for one-off searches; `-o keys|json` output; no picker when stdout is not a terminal.
- Query names match case-insensitively or by unique prefix. When nothing matches, the error lists the available names.
- Missing config is detected on any command, and the user is offered setup.
- `configure` validates each step as it is entered: it adds `https://` if missing, tests the token right after it is entered, and lets the user pick the project from a list.
- First-time setup saves once, at the end, and adds three starter queries.
- The interactive menu loops: after create, query, or configure, the user returns to the menu instead of exiting.
- `auth create` becomes the credential step of `configure`, so they share validation and messages. All "not configured" errors point to `jiractl configure`.

Depends on `fix-data-loss-and-silent-failures`.

## Capabilities

### New Capabilities
- `issue-query`: running saved and ad-hoc queries, browsing results, and acting on issues.
- `main-menu`: the interactive root menu and first-run detection.

### Modified Capabilities
- `configure-command`: per-step validation, project picker, starter queries.
- `credentials`: one credential flow shared by `configure` and `auth create`.

## Impact

- Code: `internal/cmd/{query,root,configure,auth}.go`, `internal/config/config.go` (prefix matching), `internal/jira/client.go` (transitions, assign, comment, project list).
- Clipboard: `github.com/atotto/clipboard` (new dependency). Browser: `pkg/browser` or `open`/`xdg-open`.
- go-fuzzyfinder already supports `WithPreviewWindow`.
