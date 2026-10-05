## Context

All ~40 interactive call sites go through eight helpers in `internal/cmd/prompt.go` plus one direct `fuzzyfinder.Find` (the query browser):

| Helper | Today | Used by |
| --- | --- | --- |
| `fzfSelect` | go-fuzzyfinder, full screen | menu, pickers (type, epic, project, query, actions, transitions) |
| `promptText(WithDefault)` | readline | summary, server, username, search, edits |
| `promptMultilineText` | readline loop, `.`/Ctrl+D/`:e` | description, comment |
| `promptConfirm` / `promptChoice` | readline | resume draft, save default, `Create? [Y/e/d/n]` |
| `readSecret` | `term.ReadPassword` + SIGINT restore | token |
| `openEditor` | `os/exec` | `d`, `:e` |
| `pending.wait` spinner | hand-rolled `\r` frames | prefetch waits |
| `waitForEnter`, `menuStatus` | readline / picker header | errors in the menu |

That seam lets Phase 1 swap implementations without touching the flows.

## Goals / Non-Goals

**Goals:**
- One UI toolkit, one look.
- Live epic search.
- Results and errors stay visible.
- Behavior and exit codes unchanged.
- Each phase ships independently.

**Non-Goals:**
- Mouse support.
- Theming config.
- Full-screen dashboards.
- Changing any non-interactive output.

## Decisions

### Bubble Tea v2, not v1
v2 is stable (2.0.x) and v1 gets maintenance only. v2 also brings proper Esc/key disambiguation (keyboard enhancements), a declarative `View`, and `tea.WithOutput`. Bubbles v2 and Huh v2 match it.

### Render on stderr; never start without a TTY
Every program runs with `tea.WithOutput(os.Stderr)` and `tea.WithInput(os.Stdin)`, so stdout keeps carrying only results (`KEY=$(jiractl create …)` keeps working). Helpers check `isInteractive()` first and fail with the existing "required: pass -x" errors instead of starting a program.

### Inline for prompts, alt screen only for the app shell
Phase 1 prompts and pickers render inline with a bounded height (like `fzf --height 40%`), so earlier output stays visible and scrollback keeps what happened. Only the Phase 3 app shell uses the alt screen, and it shows results in its own status bar.

### Phase 1: component per helper, same signatures
`internal/tui` exposes `Select(items, opts) (int, error)`, `Input`, `Secret`, `Confirm`, `Choice`, `Textarea`, and `Spin(label, fn)`. Each is a tiny `tea.Model` run by its own `tea.Program`. `prompt.go` becomes thin wrappers, so call sites don't change.
- Fuzzy matching: `sahilm/fuzzy` (what bubbles/list uses), treating space-separated words as AND terms. This fixes the "Change default" bug seen in tests.
- Secret: `textinput` with `EchoPassword`. Bubble Tea restores the terminal on exit, panic and Ctrl+C, so the custom SIGINT handler goes away.
- Multiline: `textarea`. Enter inserts a newline, **Ctrl+D finishes**, and Ctrl+E opens `$EDITOR` through `tea.ExecProcess`. The `.` terminator and `:e` are dropped: a line containing only `.` is now ordinary text. Bubbles' textarea binds Ctrl+D to "delete character forward" by default, so that binding is moved to the Delete key only. The prompt hint reads `Ctrl+D to finish · Ctrl+E for $EDITOR · Esc to cancel`.
- Cancel: Esc and Ctrl+C return `ErrCancelled`, so exit 130 behavior is unchanged. Pickers documented as optional keep "Esc = skip".

### Phase 2: async checks in our own components (not Huh)
Huh v2 runs a field's `Validate` synchronously on Enter and again on blur. A network check would freeze the form without a spinner and call Jira twice per answer. Instead, `tui.Input`, `tui.Secret` and `tui.Select` take an optional `Check` that runs as a `tea.Cmd`:
- Enter starts the check and shows a spinner with a label ("Checking server…").
- A failure shows the error under the field and keeps the text for correction. After `MaxAttempts` failures the component returns the error, so configure still stops after 3 failed tries and saves nothing.
- Esc cancels and Ctrl+C stops, even while a check is running.

`configure` uses these per step: server (serverInfo), credentials (/myself, re-asking the username too on rejection), and project (issue types). The create review becomes `tui.Review`: a label/value table and the `Create? [Y/e/d/n]` keys in one program. Edits reuse the Phase 1 components.

### Phase 3: one app model for menu and query browsing
`tui.App` is a screen stack (menu → query list → actions → transition picker…) with:
- a list on the left and a preview on the right (stacked when narrower than 100 columns),
- a status bar showing default epic, last result and last error (replaces `menuStatus`/`waitForEnter`/`checkLogin` header text),
- async Jira calls as `tea.Cmd`s, so the UI never blocks; a refresh after an action updates just that row,
- live epic search: each keystroke schedules a 250 ms debounced `SearchEpics` command, and stale responses are dropped by sequence number. Skip stays the first row.

`create` and `configure` keep running as their own programs (Phase 2). The app shell suspends itself (`tea.ExecProcess`-style handoff) to run them, then resumes with their result in the status bar.

### Testing
- Unit tests drive models with `tea.KeyPressMsg` and assert on `Update` results.
- `teatest/v2` golden-file tests for each component and screen at a fixed size.
- The e2e expect suite stays the end-to-end check. Patterns are updated per phase, and the stub is unchanged.

### Spike results (task 1.1)
Run under the expect harness: the first frame appears in ~40 ms and Esc exits in ~50 ms. Terminal-capability queries don't block. Two findings:
- expect's pty reports a 0×0 window, and Bubble Tea then renders nothing. The harness sets `stty_init "rows 40 cols 120"`, and `internal/tui` sets the pty to 80×24 (TIOCSWINSZ) when it reports 0×0, because Bubble Tea ignores `WithWindowSize` when output is a terminal.
- The v2 renderer redraws only changed cells, so in-place updates arrive as fragments. e2e patterns must wait for newly drawn text (prompts, headers, list rows), not for a line edited in place.

## Risks / Trade-offs

- [Bubble Tea v2 queries the terminal (keyboard enhancements, colors) and expect's pty doesn't answer] → Programs use a short query timeout and fall back to basics. The e2e harness runs with `TERM=xterm-256color`. Verify in Phase 1 before going further.
- [Inline rendering in small terminals] → Height is capped at `min(15, rows/2)`, and the list scrolls.
- [Behavior drift during the swap] → Phase 1 keeps signatures and key semantics, and the whole e2e suite must pass before release.
- [Binary size +2–4 MB] → Acceptable for a CLI distributed through Homebrew.

## Migration Plan

Each phase is one PR, one commit and one release (v0.3.0, v0.4.0, v0.5.0, then v0.5.x for cleanup). Rolling back means reverting that phase's commit. Config and data formats don't change, so mixed versions are safe.

## Open Questions

- Colors: a fixed palette, or follow the terminal's ANSI colors only (safer with odd themes)? The current proposal is ANSI 16 colors only.
