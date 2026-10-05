## 1. Client

- [ ] 1.1 Add `SearchEpics(project, text)` building the keyword and key JQL, with a fallback without wildcards
- [ ] 1.2 Add `CheckEpic(key)` → ok / not found / not an epic
- [ ] 1.3 Add `APIError.HasField(name)` to detect parent or Epic Link rejections

## 2. Picker

- [ ] 2.1 Epic search picker with fuzzyfinder hot reload, debounced server search, a fixed Skip row, and `[done]` markers
- [ ] 2.2 Use it for the normal epic choice (no default) as well as for recovery

## 3. Create flow

- [ ] 3.1 Background epic check at start; run recovery after the summary and description prompts when it fails
- [ ] 3.2 Run recovery on a parent or epic rejection and resend, at most 2 times
- [ ] 3.3 Offer to save a new default epic (`[y/N]`)
- [ ] 3.4 Add `--no-epic`; in non-interactive mode, exit 1 with up to 5 candidates

## 4. Tests and docs

- [ ] 4.1 Unit tests: JQL building, HasField, non-interactive error text
- [ ] 4.2 Manual check: delete or rename the default epic, then create through search, through Skip, and after a rejection
- [ ] 4.3 README: epic search, Skip, `--no-epic`
