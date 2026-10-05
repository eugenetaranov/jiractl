# jiractl

A command-line interface for Jira with interactive menus, secure credential storage, and saved queries.

## Features

- Interactive menus for creating issues and running queries
- Secure credential storage using system keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- TOML configuration with saved JQL queries
- Issue defaults for faster ticket creation, including custom fields for required project-specific values

## Installation

### Quick Install (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/eugenetaranov/jiractl/main/install.sh | bash
```

### Homebrew (macOS/Linux)

```bash
brew tap eugenetaranov/tap
brew install jiractl
```

### From Source

```bash
git clone https://github.com/eugenetaranov/jiractl.git
cd jiractl
task build
task install  # copies to /usr/local/bin
```

### Go Install

```bash
go install github.com/eugenetaranov/jiractl/cmd/jiractl@latest
```

### Pre-built Binaries

Download from the [releases page](https://github.com/eugenetaranov/jiractl/releases).

## Quick Start

```bash
# Configure server and credentials
jiractl configure

# Run interactive menu
jiractl

# Or use commands directly
jiractl create        # Create a new issue
jiractl query         # Select and run a saved query
jiractl query "My Open Issues"  # Run a specific query
```

## Commands

### `jiractl`

Launches an interactive menu with options to create issues, run queries, or configure settings. After each action you're back in the menu (the result of the last action is shown in its header); Esc or Exit leaves.

Before setup, any interactive command offers to run `jiractl configure` first and then continues with what you asked for.

The menu header always shows your default epic, e.g. `Default epic: OPS-40 DevOps k8s cluster upgrade`, `Default epic: none`, or `OPS-12 (not found)` / `(done)` when it can no longer be used. **Change default epic** picks a new one from the open epics (or `Search all epics…`), or `None: no default epic` to clear it. It only changes `issue_defaults.epic_link`; Esc leaves it as is. The create review also marks an epic that came from your config as `(default)`.

### `jiractl configure`

Interactive setup. Each step is checked as soon as you enter it:

1. **Server URL**: `https://` is added if you leave it out, and the server must answer as Jira.
2. **Username and API token** (generate one at https://id.atlassian.com/manage-profile/security/api-tokens): checked against Jira right away; a rejected token is asked again.
3. **Project**: picked from the projects you can see (type to filter by key or name).
4. **Default issue type and epic**: optional; Esc keeps the current value.

Nothing is saved until every step has passed. After three failures on a step, `configure` exits with status 1 and your existing setup is left untouched. On first setup, three starter queries are added: `mine`, `recent` and `unassigned`.

Saving only changes the keys whose values changed, so comments and formatting in `~/.jiractl.toml` are kept. The file is written atomically with `0600` permissions.

### `jiractl create`

Create a new Jira issue. Issue types, epics and field names are fetched in the background while you type, so the prompts don't stall.

Interactively, you're asked for the summary, description and epic. The issue type is asked only when `issue_defaults.issue_type` is not set, and the epic only when `issue_defaults.epic_link` is not set.

The description can have several paragraphs: Enter starts a new line, **Ctrl+D** finishes, and Ctrl+E continues in `$EDITOR`. Comments work the same way.

Before anything is sent, every field is listed with its Jira name (`Story Points: 3`, not `customfield_10016`), followed by `Create? [Y/e/d/n]`:

| Key | Action |
| --- | --- |
| Enter / `y` | Create the issue |
| `e` | Edit a field: type, summary, description, epic, assignee, component, labels, any custom field, or add one |
| `d` | Edit the description in `$EDITOR` |
| `n` | Save a draft and cancel |

If you answer no, or Jira rejects the issue, what you typed is saved as a draft (`~/.local/state/jiractl/draft.json`) and the next `jiractl create` offers to resume it.

If the epic (from `issue_defaults.epic_link`, a draft, or your choice) doesn't exist or isn't an epic, `create` keeps what you typed and asks you to search for another one: type words such as `devops k8s` or a key, then pick from the matching epics, or choose `Skip: create without epic`. If the missing epic was your default, you're offered to save the new one as default. The epic picker also has a `Search all epics…` row for epics beyond the most recent ones.

#### Scripting

| Flag | Meaning |
| --- | --- |
| `-s, --summary` | Summary |
| `-t, --type` | Issue type (case-insensitive; default `issue_defaults.issue_type`) |
| `-e, --epic` | Epic key (default `issue_defaults.epic_link`) |
| `--no-epic` | No epic, even if a default is set |
| `-d, --description` | Description; `-d -` reads it from stdin |
| `-F, --field name=value` | Set a field by name or `customfield_` ID; repeatable |
| `-y, --yes` | No prompts and no confirmation |

With `-y`, or when stdin is not a terminal, nothing is asked. Missing required values are an error, and only the new key is printed to stdout (the URL goes to stderr):

```bash
KEY=$(git log -1 --format=%B | jiractl create -s "Fix login" -t bug -d - -F "Story Points=3" -y)
```

### `jiractl inspect <issue-url-or-key>`

Print an issue's full JSON, including every `customfield_XXXXX` with its name. Useful for finding the IDs and value formats for `[issue_defaults.custom_fields]`:

```bash
jiractl inspect PROJ-123
jiractl inspect https://yourcompany.atlassian.net/browse/PROJ-123
```

### `jiractl query [name]`

Run a saved JQL query. Without a name, shows a menu of available queries. Names match exactly, case-insensitively, or by unique prefix (`jiractl query rec` runs `recent`); an unknown name lists the available ones.

Results are listed In Progress first, then To Do, then Done (any status in Jira's done category, such as "Completed" or "Closed"), with the most recently updated first in each group; each row shows how long ago it was updated (`3h`, `2d`, `4mo`). Epic pickers use the same order. `-o keys` and `-o json` keep Jira's order, so your JQL `ORDER BY` still applies there.

Results open in a picker with the highlighted issue's details in a preview pane. Enter opens an actions menu:

| Action | What it does |
| --- | --- |
| Open in browser | Opens the issue page |
| Copy key | Copies the key to the clipboard (or prints it when no clipboard is available) |
| Transition | Picks one of the issue's transitions |
| Assign to me | Assigns the issue to you |
| Comment | Adds a comment (same input as descriptions) |
| Back | Returns to the results |

After each action you're back in the results; Esc leaves.

```bash
jiractl query --jql 'project = ${project} AND labels = urgent'   # one-off JQL
jiractl query mine -o keys      # one key per line
jiractl query mine -o json      # [{"key", "summary", "status", "assignee"}, ...]
jiractl query mine | grep Bug   # piped: plain table, no picker
```

### `jiractl doctor`

Checks that jiractl is ready to create issues and run queries, and says what to fix:

```
[ok]   Authentication      Eugene <eugene@example.com>
[ok]   Project             OPS (5 issue types)
[warn] Transition issues   missing TRANSITION_ISSUES in OPS; 'Transition' won't work
  → ask a Jira admin for the "Transition issues" permission in OPS
[fail] Default component   component "Backnd" doesn't exist in OPS
  → set issue_defaults.component = "Backend"
[fail] Saved queries       recent: Error in the JQL Query: Expecting a date but got '-7x'.
  → fix the jql of these [[queries]] in ~/.jiractl.toml
```

It checks the config file (syntax, `0600` permissions, unknown keys), credentials, server, authentication, project, your permissions (browse, create, assign, transition, comment), the default issue type, epic, component, assignee and custom fields (on the create screen, with allowed values, plus required fields that have no default), and the JQL of every saved query. Checks that depend on a failed one are skipped.

Problems that would make `create` or `query` fail are failures (exit 1); problems that only disable one action are warnings (exit 0). `--json` prints `[{"id", "title", "status", "detail", "hint"}]`. `configure` runs the same checks when it finishes, and `jiractl auth test` runs the credential checks.

### `jiractl auth`

Manage authentication credentials:

```bash
jiractl auth list    # Show stored credentials
jiractl auth create  # Create/update credentials (checked against Jira before saving)
jiractl auth delete  # Remove credentials
jiractl auth test    # Check config, credentials, server and authentication
```

## Configuration

Configuration is stored in `~/.jiractl.toml`. Credentials are stored securely in the system keyring.

### Example Configuration

```toml
server = "https://yourcompany.atlassian.net"
project = "PROJ"

[issue_defaults]
assignee = "john.doe@example.com"   # email, name or account ID; resolved to an account ID
component = "Backend"
issue_type = "Task"
labels = ["team-alpha"]

# Custom fields applied to every created issue. Useful for required fields
# like Work Allocation that your Jira project enforces. Keys are the Jira
# custom field IDs (found in the error message when a create fails, or in
# the field configuration). Values that look like JSON are parsed as JSON
# so select-list fields can use the {"value": "..."} form; everything else
# is sent as a plain string.
[issue_defaults.custom_fields]
customfield_15838 = '{"value": "Operations"}'
customfield_10001 = "some text value"

# My assigned open issues
[[queries]]
name = "My Open Issues"
jql = "project = ${project} AND assignee = currentUser() AND status != Done ORDER BY updated DESC"
limit = 50

# Issues I'm watching
[[queries]]
name = "Watching"
jql = "watcher = currentUser() AND status != Done ORDER BY updated DESC"
limit = 30

# Recently updated in project
[[queries]]
name = "Recent Updates"
jql = "project = ${project} ORDER BY updated DESC"
limit = 20

# High priority bugs
[[queries]]
name = "Critical Bugs"
jql = "project = ${project} AND type = Bug AND priority in (Highest, High) AND status != Done"
limit = 50

# Sprint backlog
[[queries]]
name = "Current Sprint"
jql = "project = ${project} AND sprint in openSprints() ORDER BY rank ASC"
limit = 100

# Unassigned issues
[[queries]]
name = "Unassigned"
jql = "project = ${project} AND assignee is EMPTY AND status != Done ORDER BY created DESC"
limit = 30

# Created this week
[[queries]]
name = "Created This Week"
jql = "project = ${project} AND created >= startOfWeek() ORDER BY created DESC"
limit = 50

# Issues mentioning me in comments
[[queries]]
name = "Mentioned"
jql = "project = ${project} AND (text ~ currentUser() OR comment ~ currentUser()) ORDER BY updated DESC"
limit = 30

# Blocked issues
[[queries]]
name = "Blocked"
jql = "project = ${project} AND status = Blocked ORDER BY priority DESC"
limit = 50

# Due soon
[[queries]]
name = "Due This Week"
jql = "project = ${project} AND due <= endOfWeek() AND due >= startOfDay() AND status != Done ORDER BY due ASC"
limit = 30
```

### Jira Server / Data Center

`configure` detects the deployment type from the server and stores it as `deployment = "server"` (or `"cloud"`; a config without the key is treated as Cloud). On Server/Data Center (8.14+):

- you sign in with a **personal access token** (Profile → Personal Access Tokens); no username is needed,
- searches use `/rest/api/2/search`,
- `issue_defaults.assignee` is sent as a username,
- epics are linked through `parent` when the create screen has it, otherwise through the Epic Link custom field (`com.pyxis.greenhopper.jira:gh-epic-link`). Set `issue_defaults.epic_field` to force either (`"parent"` or the field ID, e.g. `"customfield_10014"`). If neither is available, the issue is created without an epic and a warning says why.

`jiractl doctor` warns when the server's actual type doesn't match `deployment` in the config.

### Query Variables

- `${project}` - Replaced with the configured project key

## Flags

- `--debug` - Enable debug output
- `-v, --version` - Show version information (works on every command)
- `-h, --help` - Show help

Unknown keys in `~/.jiractl.toml` are reported as warnings on stderr, so a typo such as `asignee` doesn't go unnoticed.

## Keys

Prompts and lists are drawn inline below your earlier output (on stderr), so nothing you've seen is cleared.

| Key | In lists | In text prompts |
| --- | --- | --- |
| type | filter; space-separated words must all match | edit |
| ↑ ↓, Ctrl+P / Ctrl+N | move | |
| Enter | choose | accept (a new line in descriptions) |
| Ctrl+D | | finish a description or comment |
| Ctrl+E | | continue a description in `$EDITOR` |
| Esc | back / skip (optional pickers) / cancel | cancel |
| Ctrl+C | stop jiractl (exit 130) | stop jiractl (exit 130) |

Colors use your terminal's own 16-color palette; set `NO_COLOR` to turn them off.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | Error (the message is printed once on stderr) |
| 130 | Cancelled with Ctrl+C or Esc |

Warnings, progress messages and prompts go to stderr, so stdout can be piped.

Errors come with a hint on what to do, e.g. an expired API token:

```
Error: query "mine" failed: Jira rejected your credentials (401): the API token is wrong or has expired
  → Create a new API token at https://id.atlassian.com/manage-profile/security/api-tokens, then run 'jiractl configure' (or pick Configure in the menu).
```

In the menu, a failed action shows the same message and waits for Enter instead of returning to the menu straight away, and the menu header warns when Jira rejects your stored token.

## Testing

```bash
task test         # unit tests
task test-e2e     # end-to-end tests against a local Jira stub (needs expect, python3)
task lint         # golangci-lint
```

## Building

Development tasks use [Task](https://taskfile.dev) (`brew install go-task`); `task` lists them all.

```bash
task build         # Build for current platform (bin/jiractl)
task build-all     # Build for linux/darwin, amd64/arm64
task run -- query mine   # go run with arguments
task clean         # Clean build artifacts
```

## Releasing

```bash
task release                 # tag and push (suggests the next patch version); CI publishes
task release TAG=v0.3.0      # explicit tag
task upgrade-local           # wait for the release, check the Homebrew formula, brew upgrade
task upgrade-local TAG=v0.3.0
```

`upgrade-local` waits for the tag's Release workflow, checks that the GitHub release and the `eugenetaranov/homebrew-tap` formula are at that version, then installs or upgrades `jiractl` with Homebrew and verifies `jiractl --version`. It needs the `gh` CLI; without Homebrew it only runs the checks. It warns when another `jiractl` earlier on your `PATH` (such as `~/go/bin/jiractl`) shadows the Homebrew one.

## License

MIT
