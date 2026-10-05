## Context

Validation is scattered today: `auth test` checks `/myself`, `configure` checks the connection, and nothing checks permissions, defaults, or saved JQL.

## Goals / Non-Goals

**Goals:** one command that answers "will create and query work?"; reuse the same checks in `configure`.

**Non-Goals:** automatically fixing problems; checking Jira workflows or field-level security beyond createmeta.

## Decisions

### Check registry
```go
type Check struct {
    ID, Title string
    DependsOn []string
    Run func(ctx *Ctx) Result  // Status: OK | Warn | Fail | Skip; Detail; Hint
}
```
Checks run in order. If a dependency is not OK or Warn, the check is marked Skip with the reason "requires <title>". `Ctx` carries the config, the client, and cached lookups (the createmeta and permissions responses).

### Checks and severity
| ID | Check | Fail / Warn |
|---|---|---|
| config | file exists, parses | Fail |
| config.perms | mode is 0600 | Warn |
| config.keys | unknown keys | Warn |
| creds | username and token present in keyring | Fail |
| server | `serverInfo` reachable; reports version and deployment type | Fail |
| auth | `/myself` returns 200; shows the display name | Fail |
| project | project exists and is visible | Fail |
| perm.browse / perm.create | `mypermissions` BROWSE_PROJECTS, CREATE_ISSUES | Fail |
| perm.assign / transition / comment | ASSIGN_ISSUES, TRANSITION_ISSUES, ADD_COMMENTS | Warn |
| default.type | issue type exists in project | Fail |
| default.epic | epic exists and is unresolved | Warn |
| default.component | component exists | Fail |
| default.assignee | resolves to exactly one assignable user | Fail |
| default.fields | every custom field is on the create screen for the default type, and allowed values match for select fields | Fail |
| queries | every saved JQL parses (`POST /rest/api/3/jql/parse`, with `${project}` expanded) | Fail per query |

Fail severity is for problems that make `create` or `query` fail. Warn severity is for problems that only disable one action.

### Output
Human output: `✓ Authentication  — Eugene (eugene@…)`. On Warn or Fail, an indented `→ hint` line follows. When stdout is not a TTY or `NO_COLOR` is set, plain ASCII markers are used (`[ok] [warn] [fail] [skip]`). `--json` prints `[{id,title,status,detail,hint}]`. The exit code is 1 if any check is Fail.

### Timeouts
Every request to Jira has a 30 s timeout (set on the HTTP client, so all commands benefit). The project-scoped checks (permissions, default epic, component, assignee, custom fields, JQL) run in parallel after `project` passes. The output stays in registry order.

### Required fields
Besides validating configured custom fields, `default.fields` warns about fields the create screen requires that have no default value and aren't covered by `issue_defaults`, because `create` would fail on them unless they're passed with `-F`.

### Reuse in configure
After it saves, `configure` runs the registry, minus `config.*`, and prints the result. Failures there don't change the exit code, because configure already validated what it needs.

## Risks / Trade-offs

- [`mypermissions` without `projectKey` returns global permissions only] → Always pass `projectKey`.
- [The createmeta API differs between Cloud and DC] → Wrap it in `client.CreateMeta`. The deployment-specific variant comes with `support-jira-server`.
- [The JQL parse endpoint differs on DC (none before 8.x)] → On 404, fall back to running the query with `maxResults=0`.
