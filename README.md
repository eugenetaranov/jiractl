# jiractl

A command-line interface for Jira with interactive menus, secure credential storage, and saved queries.

## Features

- Interactive menus for creating issues and running queries
- Secure credential storage using system keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- TOML configuration with saved JQL queries
- Issue defaults for faster ticket creation

## Installation

### From Source

```bash
git clone https://github.com/e/jiractl.git
cd jiractl
make build
make install  # copies to /usr/local/bin
```

### Pre-built Binaries

Download from the [releases page](https://github.com/e/jiractl/releases).

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

### `jiractl create`

Interactively create a new Jira issue. Prompts for issue type, summary, and description.

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
assignee = "john.doe"
component = "Backend"
issue_type = "Task"
labels = ["team-alpha"]

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
- `-v, --version` - Show version information
- `-h, --help` - Show help

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
