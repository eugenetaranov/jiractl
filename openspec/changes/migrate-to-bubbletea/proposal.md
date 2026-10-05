## Why

The interactive UI is built from three unrelated libraries: go-fuzzyfinder (full-screen pickers on tcell), chzyer/readline (line prompts) and `term.ReadPassword` (secrets). They don't share styling or key handling, and each prompt takes over or releases the terminal on its own. That causes the rough edges we've hit:

- Epic search can't update as you type, because go-fuzzyfinder doesn't expose the query.
- Pickers clear the screen, so results and errors vanish (the `waitForEnter` and menu-header workarounds).
- The picker's fuzzy filter mishandles spaces.
- There's no shared status line, layout or theme.

Bubble Tea v2 (stable, `charm.land/bubbletea/v2`) with Bubbles and Huh gives one event loop, composable components and async commands. That makes live search, inline errors and a split list/preview view straightforward.

## What Changes

Delivered in four phases. Each phase is releasable on its own, and nothing outside the interactive UI changes.

1. **Foundation:** a new `internal/tui` package re-implements the prompt helpers (select, text, secret, confirm, choice, multiline, spinner) with Bubble Tea, behind today's function signatures. Prompts render inline on stderr instead of full-screen. readline and go-fuzzyfinder are removed.
2. **Forms:** `configure` and the create review use Huh forms. Each field is validated as it is entered, with async checks for the server, token and project, and errors are shown under the field.
3. **App shell:** the main menu and query browser become one long-running program, with a split list/preview view, an actions overlay, and a status bar for results and errors (`menuStatus` and `waitForEnter` go away). **Epic search updates as you type.**
4. **Cleanup:** remove the leftover helpers, add model tests, update docs.

Unchanged: every non-interactive path (`-y`, `-o`, pipes), exit codes (0/1/130), stdout carrying only results, flags and config.

## Capabilities

### New Capabilities
- `terminal-ui`: shared interaction contract for every interactive screen: keys, cancel semantics, output stream, resizing, color.

### Modified Capabilities
- `epic-selection`: epic search filters live from Jira as you type, with no separate search prompt.
- `main-menu`: action results and errors are shown in a status bar, not by pausing.

## Impact

- New dependencies: `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/huh/v2`, `charm.land/lipgloss/v2`, `github.com/sahilm/fuzzy`; tests use `github.com/charmbracelet/x/exp/teatest/v2`.
- Removed: `github.com/ktr0731/go-fuzzyfinder` (and tcell/termbox), `github.com/chzyer/readline`.
- Code: new `internal/tui`; `internal/cmd/{prompt,root,query,create,configure,epic,default_epic,errors}.go` move onto it.
- e2e suite: expect patterns change where screens change. Flows and assertions stay.
- Binary size grows by roughly 2–4 MB.
