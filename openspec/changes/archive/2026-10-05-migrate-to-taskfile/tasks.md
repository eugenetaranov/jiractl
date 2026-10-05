## 1. Taskfile

- [x] 1.1 Add `Taskfile.yml` modeled on tack: vars, default, build, build-all, test, test-coverage, lint, clean, run, install, deps
- [x] 1.2 Add release, release-dry-run, release-snapshot, release-check (release copied from tack)
- [x] 1.3 Add the `upgrade-local` task calling `scripts/upgrade-local.sh {{.TAG}}`
- [x] 1.4 Delete the `Makefile`

## 2. Upgrade script

- [x] 2.1 Port `../tack/scripts/upgrade-local.sh` with the jiractl repo, tap, and formula values; make it executable
- [x] 2.2 Fix version parsing for the `jiractl <ver> (commit: …)` format
- [x] 2.3 Add the PATH-shadowing check against `$(brew --prefix)/bin/jiractl`

## 3. CI and docs

- [x] 3.1 `ci.yaml`: install task (`arduino/setup-task`) and run `task build`
- [x] 3.2 README: replace `make` commands; document `task release` and `task upgrade-local`

## 4. Verify

- [x] 4.1 Check that `task build` and `task test` pass, `task --list` shows every task, and `bash -n scripts/upgrade-local.sh` is clean
- [x] 4.2 Run `task upgrade-local` after the release
