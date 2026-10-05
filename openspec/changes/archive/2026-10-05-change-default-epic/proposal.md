## Why

Changing the default epic today means re-running all of `configure` or editing `~/.jiractl.toml` by hand. Users also forget which default is set, because `create` applies it without asking. The default should always be visible and take a few keystrokes to change.

## What Changes

- The main menu shows the current default epic in its header: `Default epic: OPS-40 DevOps k8s cluster upgrade`, or `Default epic: none`.
- The create review screen always shows the epic line, marked `(default)` when it came from config.
- New main-menu entry `Change default epic`. It opens the epic search from `recover-missing-epic`, with `None: no default epic` as the first row. It saves only `issue_defaults.epic_link` and returns to the menu.
- If the default epic can't be loaded (missing or resolved), the header says so: `Default epic: OPS-12 (not found)` or `OPS-12 (done)`.

Not included: a public `jiractl epic` subcommand and the offer to change the default during `create`. Both were left out on purpose (see the UX brief).

## Capabilities

### New Capabilities
- `default-epic`: showing the default epic and changing it from the main menu.

### Modified Capabilities
<!-- none -->

## Impact

- Code: `internal/cmd/root.go` (menu header and entry), `internal/cmd/create.go` (review line), a new `internal/cmd/epic.go` for the change action.
- Depends on `recover-missing-epic` (epic search picker) and on the config patcher from `fix-data-loss-and-silent-failures`. Benefits from the menu loop in `redesign-query-and-first-run`. Without it, the entry exits after saving.
