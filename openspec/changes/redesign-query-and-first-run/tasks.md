## 1. Client additions

- [ ] 1.1 Extend the search fields to cover the preview data
- [ ] 1.2 Add `GetTransitions`, `DoTransition`, `AssignToMe` (with cached `/myself`), `AddComment`, `ListProjects`, `ServerInfo`

## 2. Query

- [ ] 2.1 Add `Config.FindQuery` (exact → case-insensitive → unique prefix) and errors that list candidates
- [ ] 2.2 Add `--jql` and `-o keys|json`; skip the picker when stdout is not a TTY
- [ ] 2.3 Results picker with `WithPreviewWindow`, rendered with textutil
- [ ] 2.4 Actions menu and browse loop; refresh the row after transition, assign, or comment
- [ ] 2.5 Clipboard and browser helpers with fallbacks

## 3. Configure and credentials

- [ ] 3.1 Add `promptCredentials(server)`, shared by `configure` and `auth create`
- [ ] 3.2 Per-step validation: scheme normalization and `serverInfo`, `/myself`, project picker
- [ ] 3.3 Save once at the end; add starter queries when `queries` is empty
- [ ] 3.4 Make all "not configured" errors point to `jiractl configure`

## 4. Menu and first run

- [ ] 4.1 Add `ErrNotConfigured`, an offer-setup prompt in a TTY, then continue with the original command
- [ ] 4.2 Loop the menu; show errors and cancels inside actions and return to the menu

## 5. Docs and tests

- [ ] 5.1 Unit tests: `FindQuery`, `-o json` shape, starter query insertion
- [ ] 5.2 README: query actions, `--jql`, `-o`, first-run flow
