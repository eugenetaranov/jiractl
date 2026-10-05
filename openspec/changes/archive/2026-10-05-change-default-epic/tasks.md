## 1. Lookup and display

- [x] 1.1 Background default-epic lookup with a 1.5 s timeout and a process cache (key, summary, state: ok / not found / done)
- [x] 1.2 Menu header: `Default epic: …` line, with `none`, `(not found)`, and `(done)` variants
- [x] 1.3 Create review: always show the Epic line; mark `(default)`; `(none)` when empty

## 2. Change action

- [x] 2.1 Add `changeDefaultEpic` using the epic search picker, with a `None` row and the current default preselected
- [x] 2.2 Save or remove only `epic_link` through the config patcher; update the cache; print a confirmation to stderr
- [x] 2.3 Add the `Change default epic` menu entry; Esc returns without changes

## 3. Tests and docs

- [x] 3.1 Unit tests: header text variants, patcher set and remove of `epic_link` with comments preserved
- [x] 3.2 README: menu entry and header line
