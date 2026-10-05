## ADDED Requirements

### Requirement: Taskfile replaces Makefile
The repository SHALL provide `Taskfile.yml` with tasks equivalent to every former Makefile target, and SHALL NOT contain a `Makefile`.

#### Scenario: Build
- **WHEN** a developer runs `task build`
- **THEN** `bin/jiractl` is built, and `bin/jiractl --version` shows the git-described version and commit

#### Scenario: Task list
- **WHEN** a developer runs `task`
- **THEN** all tasks are listed with descriptions

### Requirement: Release tagging
`task release` SHALL create and push an annotated tag. When `TAG` is not given, it SHALL suggest the next patch version.

#### Scenario: Suggested tag
- **WHEN** the latest tag is `v0.4.2` and the developer presses Enter at the prompt
- **THEN** tag `v0.4.3` is created and pushed

### Requirement: Local upgrade from release
`task upgrade-local [TAG=vX.Y.Z]` SHALL wait for the Release workflow for that tag (default: latest tag). It SHALL then verify that the GitHub release exists and the Homebrew tap formula has that version, install or upgrade `jiractl` through Homebrew, and verify the installed version.

#### Scenario: Release in progress
- **WHEN** the Release workflow for the tag is still running
- **THEN** the task waits for it, then upgrades the local install, and reports `Local jiractl is <version>`

#### Scenario: Workflow failed
- **WHEN** the Release workflow concluded with failure
- **THEN** the task exits non-zero and prints the run URL

#### Scenario: Tap not updated
- **WHEN** the tap formula version differs from the tag
- **THEN** the task exits non-zero with `tap formula is at <x>, expected <y>`

#### Scenario: Shadowed binary
- **WHEN** after the upgrade, `jiractl` on PATH is not the Homebrew binary
- **THEN** the task prints a note naming the shadowing path

#### Scenario: No Homebrew
- **WHEN** Homebrew is not installed
- **THEN** the release checks still run, and the local upgrade is skipped with a message

### Requirement: CI and docs use task
CI and the README SHALL use `task` commands instead of `make`.

#### Scenario: CI build
- **WHEN** the CI workflow runs
- **THEN** it builds with `task build`
