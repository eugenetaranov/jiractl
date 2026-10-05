## Context

The Makefile's ldflags set `main.version`, `main.commit`, and `main.date`, which matches `cmd/jiractl/main.go` and `.goreleaser.yaml`. The Homebrew formula is published by goreleaser to `eugenetaranov/homebrew-tap` (`Formula/jiractl.rb`). The release workflow is named `Release`. Locally, `jiractl` currently resolves to `~/go/bin/jiractl`, which shadows the brew-installed copy.

## Goals / Non-Goals

**Goals:** the same tasks as the Makefile, plus `upgrade-local`; layout consistent with tack.

**Non-Goals:** changing goreleaser or the release workflow; Docker or integration tasks (jiractl has none).

## Decisions

### Taskfile layout
Copy tack's structure: the `vars` block (`BINARY`, `BUILD_DIR`, `VERSION`, `COMMIT`, `DATE`, `LDFLAGS`), descriptions on every task, and `default: task --list`. `build-all` loops over `linux/darwin × amd64/arm64` in a single `cmds` entry. `release` is copied from tack (`interactive: true`, prompt with a suggested tag). `run` uses `{{.CLI_ARGS}}`.

### upgrade-local.sh
Port tack's script, changing these values:
- `REPO=eugenetaranov/jiractl`, `TAP_REPO=eugenetaranov/homebrew-tap`, `TAP=eugenetaranov/tap`, `FORMULA=jiractl`.
- Version parsing: `jiractl --version` prints `jiractl <ver> (commit: …)`, so extract with `awk '{print $2}'` instead of `$3`. If the PR 1 change moves to cobra's version template, keep that output format so this still works.
- After the upgrade, compare `command -v jiractl` with `$(brew --prefix)/bin/jiractl`. If they differ, warn: `note: <path> shadows the Homebrew install; remove it or reorder PATH`.
Keep tack's `GH_PAGER=cat` and the non-interactive `gh` settings, and its handling when the run lookup fails.

### `install` task
Keep `cp bin/jiractl /usr/local/bin/` for parity with the Makefile. `upgrade-local` is the documented way to track releases.

## Risks / Trade-offs

- [The Homebrew tap formula lags behind the release] → The script fails with `tap formula is at X, expected Y`; re-run it after the tap update lands.
- [Contributors without `task`] → The README links to the install page, and CI uses `arduino/setup-task`.
