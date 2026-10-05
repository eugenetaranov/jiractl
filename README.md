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
make build
make install  # copies to /usr/local/bin
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

Launches an interactive menu with options to create issues, run queries, or configure settings.

### `jiractl configure`

Interactive setup that prompts for:
- Jira server URL (e.g., `https://yourcompany.atlassian.net`)
- Default project key (e.g., `PROJ`)
- Username (your email for Atlassian Cloud)
- API token (generate at https://id.atlassian.com/manage-profile/security/api-tokens)

The connection, credentials and project are tested before anything is saved. If a check fails, the failing value is asked again; after three failures `configure` exits with status 1 and your existing setup is left untouched.

Saving only changes the keys whose values changed, so comments and formatting in `~/.jiractl.toml` are kept. The file is written atomically with `0600` permissions.

### `jiractl create`

Create a new Jira issue. Issue types, epics and field names are fetched in the background while you type, so the prompts don't stall.

Interactively, you're asked for the summary, description and epic. The issue type is asked only when `issue_defaults.issue_type` is not set, and the epic only when `issue_defaults.epic_link` is not set.

The description can have several paragraphs: finish it with a line containing only `.` (or Ctrl+D), or type `:e` on its own line to continue in `$EDITOR`.

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

Run a saved JQL query. Without a name, shows a menu of available queries.

### `jiractl auth`

Manage authentication credentials:

```bash
jiractl auth list    # Show stored credentials
jiractl auth create  # Create/update credentials
jiractl auth delete  # Remove credentials
jiractl auth test    # Test connection to Jira
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

### Query Variables

- `${project}` - Replaced with the configured project key

## Flags

- `--debug` - Enable debug output
- `-v, --version` - Show version information (works on every command)
- `-h, --help` - Show help

Unknown keys in `~/.jiractl.toml` are reported as warnings on stderr, so a typo such as `asignee` doesn't go unnoticed.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | Error (the message is printed once on stderr) |
| 130 | Cancelled with Ctrl+C, Ctrl+D or Esc |

Warnings, progress messages and prompts go to stderr, so stdout can be piped.

## Testing

```bash
go test ./...          # unit tests
tests/e2e/run.sh       # end-to-end tests against a local Jira stub (needs expect, python3)
```

## Building

```bash
make build         # Build for current platform
make build-all     # Build for all platforms
make test          # Run tests
make lint          # Run linter
make clean         # Clean build artifacts
```

## License

MIT
