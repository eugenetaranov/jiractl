## Context

The menu header is a static `Select action`. The epic summary is fetched only during `create`. UX review picked two of the five options from the brief: always show the default, and a menu entry to change it.

## Goals / Non-Goals

**Goals:** the default epic is visible on every menu screen and on the create review; changing it from the menu takes three steps (open the entry, search, pick).

**Non-Goals:** a `jiractl epic` subcommand; prompting to change the default from `create`; per-project defaults.

## Decisions

### Header loading
Before drawing the menu, start a `GetIssue(epic_link)` with a 1.5 s timeout. The menu header is one line: `Default epic: … │ Select action`. If it hasn't returned, the header shows `Default epic: OPS-40` without the summary, and the menu is not delayed. The result is cached for the process. It is reused by `create`, and after a change the new value is put in the cache. Summaries are cut to 50 display columns with `textutil.Truncate`.

### Change action
`changeDefaultEpic(cfg, client)` lives in an internal function. It isn't a cobra command, so it adds nothing to the CLI surface, but another entry point can call it later. Steps:
1. A picker opens with the current default as the first row (so Enter keeps it), then `None: no default epic`, `Search all epics…` (the epic search from `recover-missing-epic`), and the open epics.
2. Pick an epic → save `epic_link = "<KEY>"`. Pick None → remove `epic_link`. Esc → no change.
3. Print `Default epic set to OPS-40 DevOps k8s` (or `Default epic cleared`) to stderr, then go back to the menu.
Only that key is written, through the line patcher.

### Review line
`Epic: OPS-40 DevOps k8s (default)` when it came from config, `Epic: OPS-41 … ` when chosen in this run, `Epic: (none)` otherwise. The line is always shown, including when there is no epic.

## Risks / Trade-offs

- [An extra request every time the menu opens] → It runs in the background with a timeout and a cache, and only when `epic_link` is set.
- [Without the menu loop (PR 3), the user can't see the updated header right away] → After saving, the confirmation line shows the new value.
