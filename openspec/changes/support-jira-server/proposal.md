## Why

jiractl assumes Jira Cloud: basic auth with email and API token, `/rest/api/3/search/jql`, and `parent` for epics. Server/Data Center users must authenticate with personal access tokens (Bearer). Their search endpoint and older company-managed Cloud projects link epics through the "Epic Link" custom field, so issues there can't be found or are created without an epic.

## What Changes

- Detect the deployment type from `/rest/api/2/serverInfo` (`deploymentType`) during configure and store it as `deployment = "cloud" | "server"`.
- Server/DC: authenticate with a PAT through `Authorization: Bearer`; username becomes optional.
- Server/DC: search through `/rest/api/2/search`.
- Epic linking: use `parent` when the project accepts it; otherwise fall back to the Epic Link custom field, found by its schema `com.pyxis.greenhopper.jira:gh-epic-link`. Override with `issue_defaults.epic_field`.
- Assignee on Server/DC is sent as `name`. Account ID resolution is Cloud-only.

## Capabilities

### New Capabilities
- `jira-server-support`: auth, search, assignee, and epic linking on Jira Server/Data Center and classic projects.

### Modified Capabilities
<!-- none -->

## Impact

- Code: `internal/jira/client.go` (transport choice, search endpoint, epic field), `internal/config/config.go` (`deployment`, `issue_defaults.epic_field`), `internal/cmd/configure.go`.
- Should be implemented after `redesign-query-and-first-run`, because it extends that change's per-step configure.
