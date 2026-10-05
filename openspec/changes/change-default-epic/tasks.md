## 1. Lookup and display

- [ ] 1.1 Background default-epic lookup with a 1.5 s timeout and a process cache (key, summary, state: ok / not found / done)
- [ ] 1.2 Menu header: `Default epic: …` line, with `none`, `(not found)`, and `(done)` variants
- [ ] 1.3 Create review: always show the Epic line; mark `(default)`; `(none)` when empty

## 2. Change action

- [ ] 2.1 Add `changeDefaultEpic` using the epic search picker, with a `None` row and the current default preselected
- [ ] 2.2 Save or remove only `epic_link` through the config patcher; update the cache; print a confirmation to stderr
- [ ] 2.3 Add the `Change default epic` menu entry; Esc returns without changes

## 3. Tests and docs

- [ ] 3.1 Unit tests: header text variants, patcher set and remove of `epic_link` with comments preserved
- [ ] 3.2 README: menu entry and header line
