## 1. Client

- [ ] 1.1 Add `ServerInfo`, `MyPermissions(project, perms)`, `CreateMeta(project, type)`, `ListComponents(project)`, `ParseJQL(queries)` (falling back to `maxResults=0` on 404)
- [ ] 1.2 Add a per-request timeout via context

## 2. Doctor package

- [ ] 2.1 Add the `Check`, `Result`, and `Ctx` types; a runner with dependency skips; parallel project-scoped checks in stable output order
- [ ] 2.2 Implement the checks from the design table, with hints and close-match suggestions
- [ ] 2.3 Renderers: TTY (✓ ! ✗ –), plain ASCII, JSON

## 3. Command and integration

- [ ] 3.1 Add `jiractl doctor` with `--json`; exit 1 on any failure
- [ ] 3.2 Run doctor (without the config checks) at the end of `configure`
- [ ] 3.3 Point `auth test` at the server and auth checks of the registry

## 4. Tests and docs

- [ ] 4.1 Unit tests with an `httptest` server: healthy, 401 cascade, missing permission, bad field, broken JQL
- [ ] 4.2 README: a doctor section with example output
