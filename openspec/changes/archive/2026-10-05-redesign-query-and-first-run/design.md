## Context

`runQuery` builds a fuzzyfinder list, then calls `showIssueDetails` once and exits. `runInteractiveMenu` runs one action and returns. `loadConfig` returns an error when the config is missing. `configure` prompts for everything first and checks it at the end. Even after `fix-data-loss-and-silent-failures`, a failure means re-entering the field at the end of the flow.

## Goals / Non-Goals

**Goals:** a browse-and-act loop for query results; a first run that ends with a working setup and useful queries; one credential path.

**Non-Goals:** editing arbitrary fields from the query view; pagination beyond `limit`; multi-profile configs.

## Decisions

### Preview pane
Use `fuzzyfinder.WithPreviewWindow`. Preview text comes from the search results, which are extended to request `issuetype,priority,assignee,reporter,labels,description`. Jira Cloud's v3 search returns descriptions as Atlassian Document Format; they are flattened to plain text before go-jira decodes the issues. That way the preview needs no extra request per item. The description is truncated to the preview height and wrapped by display width (`textutil`).

### Action loop
```
for {
  idx := picker(results)  // Esc → return nil (back to menu / exit 0)
  for {
    action := picker(actions(issue))  // Esc or "Back" → break
    run(action)  // after a transition, assign, or comment, re-fetch that issue and update the row
  }
}
```
Esc in the results list is a normal exit (status 0), not a cancel, because browsing is the purpose of this screen. This is the documented exception to the cancel rule in `cli-conventions`.

Actions:
- open: `browser.OpenURL`
- copy: `clipboard.WriteAll(key)`; if no clipboard is available, print the key
- transition: `GET /issue/{key}/transitions`, then a picker, then `POST`
- assign to me: `PUT /issue/{key}/assignee` with `accountId` from `/myself`, cached
- comment: multiline prompt from `redesign-create-flow`, or `$EDITOR`

### Non-interactive output
If `-o` is given, or stdout is not a TTY, skip the picker. `-o keys` prints one key per line. `-o json` prints a JSON array of `{key, summary, status, assignee}`. When `-o` is omitted and stdout is not a TTY, the default is a plain table, one issue per line.

### Name matching
`Config.FindQuery(name)` tries, in order: an exact match, a case-insensitive match, then a unique case-insensitive prefix. If the prefix matches several queries, the error lists them. If nothing matches, the error lists all query names.

### First-run detection
`loadConfig` returns `ErrNotConfigured`. In the root menu, and in commands run in a TTY, catch it and ask `jiractl isn't set up yet. Run setup now? [Y/n]`. Yes runs configure and then continues with the original command. Outside a TTY, it is an error: `not configured: run 'jiractl configure'`.

### Per-step configure
1. Server: add `https://` if no scheme is given, then `GET /rest/api/2/serverInfo` (no auth needed). Re-ask on failure.
2. Username and token: `auth.promptCredentials(server)` is shared with `auth create`. It tests `/myself` right away and re-asks on 401.
3. Project: a picker from `GET /rest/api/2/project/search`, with the current project preselected. Typing a key is still possible.
4. Default issue type and epic: optional pickers. Esc keeps the previous value.
5. Save everything once, using the patcher from the earlier change. If the config had no queries, append the starter queries: `mine` (assigned to me, not done), `recent` (updated in the last 7 days in the project), and `unassigned` (unassigned, not done).

`auth create` = step 2 with the server from config. If no server is configured, it runs step 1 first.

### Menu loop
`runInteractiveMenu` loops until Exit or Esc. A cancel or error inside an action prints the message and returns to the menu, and does not exit. Because the picker takes over the screen, the last action's result (for example `Created OPS-99`) is shown in the menu header.

## Risks / Trade-offs

- [The project list can be large] → fuzzy search handles it; fetch `maxResults=1000`.
- [No clipboard on headless Linux] → print the key instead and say why.
- [Adding starter queries surprises existing users] → add them only when `queries` is empty.
