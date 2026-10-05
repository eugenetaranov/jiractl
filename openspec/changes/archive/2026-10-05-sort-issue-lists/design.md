## Context

Search results already include `status` (with `statusCategory`) and `updated`, so ordering needs no extra requests.

## Decisions

- **Last update over creation date:** "recently touched" means updated; an old issue being worked on today should be near the top.
- **Category, not status name:** status names vary per workflow; `statusCategory.key` is `indeterminate` (In Progress), `new` (To Do) or `done`. Unknown categories rank with To Do; an issue with a resolution but no category ranks as Done.
- **Stable sort** keeps Jira's order for equal keys.
- **Machine output untouched:** `-o keys|json` stay in JQL order; the table, meant for people, is sorted.

## Risks / Trade-offs

- [Overrides the JQL `ORDER BY` in the picker] → Deliberate: the picker is for finding active work. Scripts that rely on JQL order use `-o keys|json`.
