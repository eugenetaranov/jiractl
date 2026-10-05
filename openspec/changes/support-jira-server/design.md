## Context

`NewClient` always uses `BasicAuthTransport`. `SearchIssues` calls `/rest/api/3/search/jql`, which exists only on Cloud. `CreateIssue` always sets `Parent`.

## Goals / Non-Goals

**Goals:** Server/DC 8.14+ works with a PAT; classic projects get an epic link.

**Non-Goals:** OAuth; basic auth with a password on Server; Jira versions before 8.14.

## Decisions

### Deployment detection
In the server step of configure, `serverInfo.deploymentType == "Cloud"` → `cloud`, anything else → `server`. The value is stored in config so later runs skip the request. A missing value is treated as `cloud`, which keeps existing configs working.

### Transport
`server` → `jira.BearerAuthTransport{Token}`. The username prompt is skipped, and `/myself` checks the token. `cloud` → basic auth as now.

### Search
A `searchPath()` helper picks the endpoint: `cloud` → `rest/api/3/search/jql`, `server` → `rest/api/2/search`. Both return `issues`.

### Epic field
Resolve it once, through createmeta for the project and issue type:
1. If `issue_defaults.epic_field` is set, use it.
2. If the create screen has `parent`, use `parent: {key}`.
3. If a field has the schema custom type `gh-epic-link`, set that field to the epic key.
4. Otherwise, warn on stderr that the epic link is not supported for this project, and create the issue without it.

## Risks / Trade-offs

- [createmeta differs between Cloud and DC (`/createmeta/{project}/issuetypes/{id}` vs the expanded form)] → Pick the endpoint based on deployment.
- [No Server instance available for testing] → Use a recorded HTTP fixture test for each deployment type.
