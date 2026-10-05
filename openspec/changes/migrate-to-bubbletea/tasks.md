## 1. Phase 1: Foundation (release v0.3.0)

- [x] 1.1 Spike: run a v2 program under the e2e expect harness; confirm terminal queries don't hang and Esc is delivered immediately
- [x] 1.2 Add `internal/tui` with stderr output, a TTY guard, an ANSI-16 style set and NO_COLOR handling
- [x] 1.3 `tui.Select`: inline fuzzy list (sahilm/fuzzy, AND terms), header, bounded height, optional-skip mode
- [x] 1.4 `tui.Input` (default, required), `tui.Secret` (EchoPassword), `tui.Confirm`, `tui.Choice`
- [x] 1.5 `tui.Textarea`: Ctrl+D finishes (rebind the default Ctrl+D delete-forward to Delete only), Ctrl+E opens `$EDITOR` via ExecProcess; drop the `.` and `:e` lines
- [x] 1.6 `tui.Spin` replacing the `pending.wait` spinner
- [x] 1.7 Switch `prompt.go` helpers to `internal/tui`; port the query browser's preview list to `tui.Select` with a preview pane
- [x] 1.8 Remove readline and go-fuzzyfinder; `go mod tidy`
- [x] 1.9 Update e2e expect patterns; whole suite green; teatest goldens per component

## 2. Phase 2: Checked fields (release v0.4.0)

- [x] 2.1 Async `Check` (spinner, inline error, MaxAttempts) for tui.Input, tui.Secret and tui.Select; `configure` steps use it
- [x] 2.2 `auth create` uses the same credentials step
- [x] 2.3 `tui.Review`: payload table and y/e/d/n keys in one program; edits reuse the Phase 1 components
- [x] 2.4 Tests: model unit tests, goldens, e2e for configure and create

## 3. Phase 3: App shell (release v0.5.0)

- [x] 3.1 `tui.App` screen stack; menu screen with a status bar (default epic, last result, last error with hint)
- [x] 3.2 Query browser screen: list + preview split (stacked under 100 columns), async refresh of a changed row
- [x] 3.3 Actions overlay: open, copy, transition, assign, comment
- [x] 3.4 Live epic search component: debounced commands, sequence numbers to drop stale responses, Skip row; used by create, change default epic and configure
- [x] 3.5 Hand off to create/configure programs and resume with their result
- [x] 3.6 Remove `menuStatus`, `waitForEnter`, and the header text from `checkLogin`

## 4. Phase 4: Cleanup (v0.5.x)

- [x] 4.1 Delete unused helpers (tui.Pause, textutil.Width); golden views for every component and the app frame
- [x] 4.2 README: keys, app layout, live epic search, NO_COLOR (verified); screenshots left out
- [x] 4.3 Close the open question on colors: the terminal's 16 ANSI colors
