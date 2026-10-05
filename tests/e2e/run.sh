#!/usr/bin/env bash
# End-to-end tests: drive the real binary through a pty (expect) against a
# local Jira stub. Uses an in-memory keyring (-tags keyringmock) and a
# throwaway HOME, so nothing on the machine is touched.
#
# Usage: tests/e2e/run.sh            (requires expect and python3)
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
WORK=$(mktemp -d)
trap 'kill $STUB_PID 2>/dev/null || true; rm -rf "$WORK"' EXIT

PORT=${PORT:-18799}
BIN=$WORK/jiractl
go build -tags keyringmock -o "$BIN" "$ROOT/cmd/jiractl"

export STUB_LOG=$WORK/creates.log
python3 "$ROOT/tests/e2e/stub.py" "$PORT" &
STUB_PID=$!
sleep 0.5

FAILED=0
pass() { echo "ok   $1"; }
fail() { echo "FAIL $1: $2"; FAILED=1; }

# new_home writes a config and prints the HOME directory.
new_home() {
  local h
  h=$(mktemp -d "$WORK/home.XXXX")
  printf '%s\n' "server = \"http://127.0.0.1:$PORT\"" 'project = "OPS"' "$@" > "$h/.jiractl.toml"
  echo "$h"
}

# run_expect HOME SCRIPT: runs jiractl with the given expect body.
run_expect() {
  local home=$1 body=$2 args=${3:-create}
  : > "$STUB_LOG"
  cat > "$WORK/s.exp" <<EXP
set timeout 10
log_file -noappend $WORK/out.log
spawn env HOME=$home XDG_STATE_HOME=$home/state JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=t $BIN $args
$body
expect eof
catch wait result
exit [lindex \$result 3]
EXP
  set +e
  expect "$WORK/s.exp" >/dev/null 2>&1
  EXIT=$?
  set -e
}

out_has() { LC_ALL=C grep -a -q -E "$1" "$WORK/out.log"; }

# --- missing default epic: search, pick, save as default
H=$(new_home '' '[issue_defaults]' 'issue_type = "Task"' 'epic_link = "OPS-12"' '# keep me')
run_expect "$H" '
expect "Summary" { send "Fix login\r" }
expect "> " { send "\r" }
expect "Search epics" { send "devops k8s\r" }
expect "1 epics match" { sleep 0.5; send "OPS-40" }
sleep 0.5
send "\r"
expect "default epic?" { send "y\r" }
expect "Create this issue?" { send "\r" }'
if grep -q '"parent": {"key": "OPS-40"}' "$STUB_LOG" && grep -q 'epic_link = "OPS-40"' "$H/.jiractl.toml" && grep -q '# keep me' "$H/.jiractl.toml"; then
  pass "missing epic: search and pick"
else
  fail "missing epic: search and pick" "$(cat "$STUB_LOG")"
fi

# --- missing default epic: skip
H=$(new_home '' '[issue_defaults]' 'issue_type = "Task"' 'epic_link = "OPS-12"')
run_expect "$H" '
expect "Summary" { send "No epic\r" }
expect "> " { send "\r" }
expect "Search epics" { send "\r" }
expect "open epics" { sleep 0.5; send "Skip" }
sleep 0.5
send "\r"
expect "Create this issue?" { send "\r" }'
if [ -s "$STUB_LOG" ] && ! grep -q parent "$STUB_LOG" && grep -q 'epic_link = "OPS-12"' "$H/.jiractl.toml"; then
  pass "missing epic: skip"
else
  fail "missing epic: skip" "$(cat "$STUB_LOG")"
fi

# --- configure: wrong project is re-asked and nothing is saved
H=$(new_home '# my comment')
cp "$H/.jiractl.toml" "$WORK/orig"
run_expect "$H" '
expect "Server URL" { send "\r" }
expect "Project Key" { send "nope\r" }
expect "Username" { send "u\r" }
expect "API Token" { send "secret\r" }
expect "Project Key" { send "nope\r" }
expect "Project Key" { send "nope\r" }' configure
if [ "$EXIT" = 1 ] && cmp -s "$WORK/orig" "$H/.jiractl.toml"; then
  pass "configure: failed validation saves nothing"
else
  fail "configure: failed validation saves nothing" "exit=$EXIT"
fi

exit $FAILED
