## 1. Shared plumbing

- [x] 1.1 Set `SilenceErrors` and `SilenceUsage` on `RootCmd`; remove the per-command `SilenceUsage` lines
- [x] 1.2 Add `ErrCancelled`; map readline interrupt/EOF and fuzzyfinder `ErrAbort` to it in every prompt helper; have `promptConfirm` return it on Ctrl+C
- [x] 1.3 In `Execute()`, print `Cancelled.` to stderr and exit 130 for `ErrCancelled`; replace the five ad-hoc cancel messages
- [x] 1.4 Make `-v/--version` a persistent root flag so it works on every command
- [x] 1.5 Send warnings, progress, and the query banner to stderr
- [x] 1.6 Add `internal/textutil` (`Truncate`, `PadRight` on go-runewidth) and use it in query results and epic pickers

## 2. Config file

- [x] 2.1 Implement the line patcher for scalar keys and table insertion, with a backup-and-rewrite fallback
- [x] 2.2 Write through a temp file, fsync, and rename, with mode 0600
- [x] 2.3 Warn on `MetaData.Undecoded()` keys
- [x] 2.4 Unit tests: comments preserved, missing table, CRLF, write failure leaves the original intact, permissions

## 3. Configure

- [x] 3.1 Add `jira.NewClientWith(server, user, token)`; make `NewClient(cfg)` wrap it
- [x] 3.2 Validate server, token, and project in memory; re-ask the failing field, up to 3 attempts; exit 1 on failure
- [x] 3.3 Save the keyring and config once, after validation; make Esc in the optional pickers keep the previous value; print an accurate summary

## 4. Credentials

- [x] 4.1 Add `readSecret()` with terminal state restore on SIGINT; use it in `configure` and `auth create`
- [x] 4.2 `auth list`: show `set (N chars)`; remove the token prefix from `--debug`

## 5. Create and defaults

- [x] 5.1 Send `components` from `issue_defaults.component`
- [x] 5.2 Resolve `assignee` to an accountId through user search, with a process cache and a clear error on ambiguity
- [x] 5.3 Change the confirm to `[Y/n]`, with Enter meaning yes
- [x] 5.4 Write a draft on decline or create failure; offer to resume it on the next `create`; delete it on success
- [x] 5.5 Add `jira.APIError`, parsing `errorMessages` and `errors`; use it in create and search; remove the duplicate prefixes

## 6. Docs and verification

- [x] 6.1 README: document `inspect`; fix the issue-type sentence for create; document exit codes 1 and 130
- [x] 6.2 Manual check: each item in the review table reproduces before the fix and is fixed after
