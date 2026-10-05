## 1. Config and detection

- [x] 1.1 Add `deployment` and `issue_defaults.epic_field` to config (and to the known-keys list)
- [x] 1.2 In the configure server step, read `deploymentType` and store it; for `server`, prompt for a PAT instead of an email and token

## 2. Client

- [x] 2.1 Choose the transport by deployment (Bearer vs Basic)
- [x] 2.2 Add `searchPath()` and use it in `SearchIssues`
- [x] 2.3 Assignee: send `name` on server, `accountId` on cloud
- [x] 2.4 Resolve the epic field through createmeta, following the override → parent → gh-epic-link → warn order

## 3. Tests and docs

- [x] 3.1 HTTP fixture tests for cloud and server: auth header, search path, epic field, assignee
- [x] 3.2 README: Server/Data Center section (PAT, `epic_field`)
