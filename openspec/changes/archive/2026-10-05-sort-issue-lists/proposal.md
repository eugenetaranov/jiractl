## Why

Lists show issues in whatever order Jira or the JQL returns them, so finished issues sit among active ones and recently touched work can be far from the cursor.

## What Changes

- Issue lists are ordered by status category (In Progress, then To Do, then Done), then by last update, newest first. Done covers every status in Jira's done category ("Done", "Completed", "Closed"…).
- Applies to the query picker, the piped table, and every epic picker (create, epic search, change default epic, configure).
- `-o keys` and `-o json` keep Jira's order, so a script's own `ORDER BY` still decides.
- Rows get an "updated" column with a short age (`5m`, `3h`, `2d`, `3w`, `4mo`); the preview shows the age too.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `issue-query`: result ordering and the updated column.
- `epic-selection`: epic lists use the same ordering.

## Impact

- New `jira.SortForDisplay`, used by `cmd/query.go` and the epic pickers; `GetEpics` orders by `updated DESC`.
- e2e stub issues gain status categories and update times.
