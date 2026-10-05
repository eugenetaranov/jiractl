## 1. Client

- [x] 1.1 Add `ServerInfo`, `MyPermissions(project, perms)`, `CreateMeta(project, type)`, `ListComponents(project)`, `ParseJQL(queries)` (falling back to `maxResults=0` on 404)
- [x] 1.2 Add a 30 s timeout to every request (http.Client.Timeout)

## 2. Doctor package

- [x] 2.1 Add the `Check`, `Result`, and `Ctx` types; a runner with dependency skips; parallel project-scoped checks in stable output order
- [x] 2.2 Implement the checks from the design table, with hints and close-match suggestions
- [x] 2.3 Renderers: TTY (✓ ! ✗ –), plain ASCII, JSON

## 3. Command and integration

- [x] 3.1 Add `jiractl doctor` with `--json`; exit 1 on any failure
- [x] 3.2 Run doctor (without the config checks) at the end of `configure`
- [x] 3.3 Point `auth test` at the server and auth checks of the registry

## 4. Tests and docs

- [x] 4.1 Unit tests with an `httptest` server: healthy, 401 cascade, missing permission, bad field, broken JQL
- [x] 4.2 README: a doctor section with example output
