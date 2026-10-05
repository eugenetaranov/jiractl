## 1. Model and client

- [ ] 1.1 Add the `issueDraft` struct and merge order (defaults → draft → flags → prompts → edits)
- [ ] 1.2 Change `client.CreateIssue` to take `issueDraft` and send exactly its fields
- [ ] 1.3 Add `client.GetFieldNames()` (`/rest/api/2/field`) returning id↔name maps

## 2. Prefetch

- [ ] 2.1 Add a `prefetch` struct with per-item wait and error; start the goroutines at the beginning of `create`
- [ ] 2.2 Add a stderr spinner after a 300 ms wait; report errors only when the item is needed

## 3. Interactive flow

- [ ] 3.1 Multiline prompt: keep blank lines; end on `.` or Ctrl+D; `:e` opens the editor
- [ ] 3.2 Add an `openEditor(initial string)` helper ($EDITOR → $VISUAL → vi, or notepad on Windows)
- [ ] 3.3 Review screen with field display names
- [ ] 3.4 `Create? [Y/e/d/n]` loop with field picker, edit prompt, and editor

## 4. Flags

- [ ] 4.1 Add `-s -t -e -d -F -y`; `-d -` reads stdin
- [ ] 4.2 Non-interactive mode when `-y` is set or stdin is not a TTY; required-value errors
- [ ] 4.3 Resolve `-F` names through the field map, with close-match suggestions on a miss
- [ ] 4.4 On success, print only the key to stdout and the URL to stderr

## 5. Docs and tests

- [ ] 5.1 Unit tests: draft merge order, `-F` parsing, multiline terminator
- [ ] 5.2 README: new flags, confirm keys, description input, scripting example
