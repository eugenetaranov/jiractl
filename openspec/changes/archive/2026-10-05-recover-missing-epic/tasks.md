## 1. Client

- [x] 1.1 Add `SearchEpics(project, text)` building the keyword and key JQL, with a fallback without wildcards
- [x] 1.2 Add `CheckEpic(key)` → ok / not found / not an epic
- [x] 1.3 Add `APIError.HasField(name)` to detect parent or Epic Link rejections

## 2. Picker

- [x] 2.1 Epic search: prompt for words, server search, picker with Skip and Search again rows and `[done]` markers
- [x] 2.2 Use it for the normal epic choice (no default) as well as for recovery

## 3. Create flow

- [x] 3.1 Background epic check at start; run recovery after the summary and description prompts when it fails
- [x] 3.2 Run recovery on a parent or epic rejection and resend, at most 2 times
- [x] 3.3 Offer to save a new default epic (`[y/N]`)
- [x] 3.4 Add `--no-epic`; in non-interactive mode, exit 1 with up to 5 candidates

## 4. Tests and docs

- [x] 4.1 Unit tests: JQL building, HasField, non-interactive error text
- [x] 4.2 End-to-end check (tests/e2e/run.sh): delete or rename the default epic, then create through search, through Skip, and after a rejection
- [x] 4.3 README: epic search, Skip, `--no-epic`
