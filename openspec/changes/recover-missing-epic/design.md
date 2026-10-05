## Context

Today, `create.go` calls `GetIssue(epicLink)` only to show the summary, and ignores the error. `CreateIssue` then sets `parent` and Jira answers 400. The interactive epic picker searches a local list of up to 100 unresolved epics.

## Goals / Non-Goals

**Goals:** a missing epic never ends the create flow in interactive mode; finding an epic by keywords.

**Non-Goals:** updating `epic_link` in config automatically (we offer it, see below); choosing non-epic parents such as stories for sub-tasks.

## Decisions

### Validate up front
Start `GetIssue(epic)` in the background when create begins. The check fails when the issue is not found (404), is not an epic, or belongs to a different project and the user lacks access. If it fails, recovery runs right after the summary and description prompts, so typed text is never lost.

### Search-as-you-type
Use go-fuzzyfinder with `WithHotReloadLock`. The item slice is updated from a debounced (250 ms) server search:
`project = P AND issuetype = Epic AND (summary ~ "w1*" AND summary ~ "w2*" OR key = "<input>") ORDER BY resolution ASC, updated DESC`, `maxResults=50`.
Unresolved epics come first; resolved ones are marked `[done]`. Each search word must match the summary, and a word that looks like a key also matches the key. Fixed rows at the top: `Skip: create without epic`, then the matches.
- Alternative: a prompt for the search text, then a picker. Simpler, but the user can't refine the search without starting over. If hot reload turns out to be unreliable, we use this instead.

### After choosing
- An epic is picked and it came from the config default: ask `Save <KEY> as default epic? [y/N]`. Yes updates `epic_link` through the config patcher.
- Skip: continue without an epic. The review screen shows `Epic: (none)`.
- Esc in the picker means the same as Skip, because the user can still cancel at the confirm step.

### Rejection after send
`jira.APIError.HasField("parent")` or a match on the Epic Link field ID triggers the same recovery, then the request is sent again. This happens at most twice, so the loop can't repeat forever.

### Non-interactive
Missing epic and no `--no-epic`: the error is `epic ABC-12 not found; candidates: ABC-40 DevOps k8s migration, …` (top 5, from a search on the old epic's summary words when available, otherwise the 5 newest open epics). Exit 1.

## Risks / Trade-offs

- [Text search syntax differs on DC (`~` is supported, wildcard behavior varies)] → If the wildcard query fails, retry with plain `summary ~ "words"`.
- [Search requests on every keystroke] → Debounce, and cache results per query.
