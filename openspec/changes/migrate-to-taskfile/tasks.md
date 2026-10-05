## 1. Taskfile

- [ ] 1.1 Add `Taskfile.yml` modeled on tack: vars, default, build, build-all, test, test-coverage, lint, clean, run, install, deps
- [ ] 1.2 Add release, release-dry-run, release-snapshot, release-check (release copied from tack)
- [ ] 1.3 Add the `upgrade-local` task calling `scripts/upgrade-local.sh {{.TAG}}`
- [ ] 1.4 Delete the `Makefile`

## 2. Upgrade script

- [ ] 2.1 Port `../tack/scripts/upgrade-local.sh` with the jiractl repo, tap, and formula values; make it executable
- [ ] 2.2 Fix version parsing for the `jiractl <ver> (commit: …)` format
- [ ] 2.3 Add the PATH-shadowing check against `$(brew --prefix)/bin/jiractl`

## 3. CI and docs

- [ ] 3.1 `ci.yaml`: install task (`arduino/setup-task`) and run `task build`
- [ ] 3.2 README: replace `make` commands; document `task release` and `task upgrade-local`

## 4. Verify

- [ ] 4.1 Check that `task build` and `task test` pass, `task --list` shows every task, and `bash -n scripts/upgrade-local.sh` is clean
- [ ] 4.2 Run `task upgrade-local` against the current latest tag
