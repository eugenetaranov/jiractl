## MODIFIED Requirements

### Requirement: Search epics by keywords
The epic search SHALL query Jira as the user types, after a pause of 250 ms, for epics in the project whose summary matches every word, or whose key matches the input. Results SHALL replace the list as they arrive, with responses to older input discarded, and unresolved epics listed first.

#### Scenario: Keyword search
- **WHEN** the user types `devops k8s` in the epic search
- **THEN** the list shows epics whose summaries contain words starting with `devops` and `k8s`, such as `OPS-40 DevOps: k8s cluster upgrade`, without pressing Enter first

#### Scenario: Search by key
- **WHEN** the user types `OPS-40`
- **THEN** OPS-40 is listed

#### Scenario: Out-of-order responses
- **WHEN** the response for `dev` arrives after the response for `devops`
- **THEN** the list keeps showing the results for `devops`
