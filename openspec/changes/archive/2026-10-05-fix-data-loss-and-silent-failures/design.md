## Context

The code is small (about 1.4k lines in `internal/`). The config uses BurntSushi/toml, which does not preserve comments on encode. The prompts use chzyer/readline, `term.ReadPassword`, and go-fuzzyfinder. Errors currently go through cobra plus `Execute()`, so each one prints twice.

## Goals / Non-Goals

**Goals:** no data loss in `configure` or `create`; no silently ignored config; consistent cancel and error behavior; output that can be piped.

**Non-Goals:** the create redesign (edit field, `$EDITOR`, flags), per-step validation and the project picker in configure, and query UX. Those are in `redesign-create-flow` and `redesign-query-and-first-run`.

## Decisions

### Comment-preserving config writes
`Config.Save` becomes `Config.Update(changes)`. It reads the existing file and patches lines in place for scalar keys (`server`, `project`, `issue_defaults.*`). It appends new `[[queries]]` blocks at the end of the file. If a key is missing, it inserts the key under the right table header, or creates the header. The result is written to `~/.jiractl.toml.tmp-<pid>` with mode 0600, fsynced, then renamed over the original.
- Alternative: `pelletier/go-toml/v2`. It does not keep comments either.
- Alternative: rewrite the whole file and keep a `.bak` copy. Simpler, but it still loses the user's formatting. We use it only as a fallback when the patcher cannot find an unambiguous place for a key, such as an inline table. The fallback warns on stderr.

### configure: test before persist
Build the `jira.Client` from in-memory values instead of the keyring. `jira.NewClientWith(server, user, token)` is a new constructor, and `NewClient(cfg)` wraps it. Call `TestConnection`, then `GetIssueTypes(project)`. On a 401 or 403, re-ask the token. On a network or URL error, re-ask the server. On a 404 from the project lookup, re-ask the project. Allow 3 attempts, then return an error, which exits 1. Write the keyring and the config only after both checks pass. Esc in the optional pickers that follow keeps the previous value, and the closing message reflects what was actually saved.

### Cancellation
Add `var ErrCancelled = errors.New("cancelled")`. Every prompt and picker maps readline's `ErrInterrupt`/EOF and fuzzyfinder's `ErrAbort` to it. `Execute()` checks with `errors.Is(err, ErrCancelled)`, prints `Cancelled.` to stderr, and exits 130. `promptConfirm` returns `ErrCancelled` on Ctrl+C instead of `false`. Set `RootCmd.SilenceErrors = true` and `SilenceUsage = true` once on the root command. Prompts are written to stderr (readline `Stdout: os.Stderr`).

### Version flag
Cobra's built-in version flag is only registered on the root command, so `jiractl query --version` would fail. Instead, `--version`/`-v` is a persistent flag on the root, handled in `PersistentPreRunE`. It prints the version and returns a sentinel error, which `Execute()` treats as success.

### Token prompt restore
Wrap `term.ReadPassword` in `readSecret()`. It saves the terminal state with `term.GetState`, installs a `signal.Notify(os.Interrupt)` handler that restores the state and returns `ErrCancelled`, and restores the state with defer. `configure` and `auth create` both use it.

### Assignee resolution
Before creating an issue, call `GET /rest/api/3/user/search?query=<assignee>`. Use the result only when exactly one active user matches (an exact email or display-name match wins among several), and send `{"accountId": ...}` as a raw field, because go-jira's `User` type always serializes an empty `Password`. On zero or several matches, return an error that lists the candidates and does not create the issue. Cache the result for the life of the process. A config value that already looks like an account ID (contains `:` or is 24+ hex characters) is sent as is.

### Unknown keys
After decoding, call `toml.MetaData.Undecoded()`. For each unknown key, print `warning: unknown config key "x" in ~/.jiractl.toml` to stderr.

### Draft on failure
When `CreateIssue` fails, write the entered fields as JSON to `$XDG_STATE_HOME/jiractl/draft.json` (default `~/.local/state/jiractl/draft.json`) with mode 0600. Print the path. On the next `create`, offer `Resume draft from <time>? [Y/n]`. Delete the draft after a successful create.

### Jira error formatting
Add `jira.APIError{Status int, Messages []string}`. It parses `errorMessages` and `errors{field: msg}` from the response body. `Error()` joins them as `field: msg; …`. The create and search paths return it without adding their own prefix, so one prefix is added at most once, in the command.

### Text width
Add `internal/textutil` with `Truncate(s, cols)` and `PadRight(s, cols)`, built on `go-runewidth`. Use it for summaries in query results and the epic picker.

## Risks / Trade-offs

- [The line patcher could misread an unusual TOML layout] → Patch only plain `key = value` lines inside known tables. Otherwise use the backup-and-rewrite fallback. Unit tests cover files with comments, missing tables, and CRLF line endings.
- [Exit code 130 breaks scripts that expected 0] → Document it in the README and changelog.
- [User search may need the "Browse users" permission] → On a 403, the error says to set an account ID directly in `assignee`.
