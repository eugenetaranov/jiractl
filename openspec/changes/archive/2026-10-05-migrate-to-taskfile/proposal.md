## Why

The repo uses a Makefile. The release recipe is a hard-to-read shell one-liner, and nothing upgrades the local install after a release is published. Sibling projects (`../tack`) already use Taskfile, and there it includes an `upgrade-local` task that waits for the release and reinstalls through Homebrew.

## What Changes

- Replace `Makefile` with `Taskfile.yml`, modeled on `../tack/Taskfile.yml`. Tasks: `default` (list), `build`, `build-all`, `test`, `test-coverage`, `lint`, `clean`, `run` (`-- args`), `install`, `deps`, `release`, `upgrade-local`, `release-dry-run`, `release-snapshot`, `release-check`.
- Add `scripts/upgrade-local.sh`, adapted from tack. It waits for the `Release` workflow for the tag, confirms that the GitHub release and the `eugenetaranov/homebrew-tap` formula are at that version, then runs `brew upgrade` or `brew install` and verifies the installed version.
- The script warns when another `jiractl` earlier on `PATH` (for example `~/go/bin/jiractl`) shadows the Homebrew binary.
- Update CI (`ci.yaml` uses `make build`) and the README to use `task`.
- **BREAKING** (dev workflow only): `make` targets are removed.

## Capabilities

### New Capabilities
- `build-tooling`: developer tasks for build, test, release, and local upgrade.

### Modified Capabilities
<!-- none -->

## Impact

- Files: remove `Makefile`; add `Taskfile.yml` and `scripts/upgrade-local.sh`; edit `.github/workflows/ci.yaml` and `README.md`.
- Developer tools: `task` (go-task), `gh` (for `upgrade-local`), and Homebrew (optional; the script skips the local upgrade without it).
