## 1. Ordering

- [x] 1.1 Add `jira.StatusRank` and `jira.SortForDisplay` (category, then updated desc, stable)
- [x] 1.2 Sort query results for the picker and table; leave keys/json in Jira order
- [x] 1.3 Sort epic lists (GetEpics with ORDER BY updated DESC, SearchEpics)

## 2. Display

- [x] 2.1 Add a short age helper and an updated column to result rows; show it in the preview

## 3. Tests

- [x] 3.1 Unit tests: ranking, ordering, age formatting
- [x] 3.2 e2e: stub issues with categories and update times; table order; JSON order unchanged
