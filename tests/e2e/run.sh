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
spawn env EDITOR=$WORK/editor.sh HOME=$home XDG_STATE_HOME=$home/state JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=t $BIN $args
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
expect "> " { send ".\r" }
expect "Search epics" { send "devops k8s\r" }
expect "1 epics match" { sleep 0.5; send "OPS-40" }
sleep 0.5
send "\r"
expect "default epic?" { send "y\r" }
expect "Create?" { send "\r" }'
if grep -q '"parent": {"key": "OPS-40"}' "$STUB_LOG" && grep -q 'epic_link = "OPS-40"' "$H/.jiractl.toml" && grep -q '# keep me' "$H/.jiractl.toml"; then
  pass "missing epic: search and pick"
else
  fail "missing epic: search and pick" "$(cat "$STUB_LOG")"
fi

# --- missing default epic: skip
H=$(new_home '' '[issue_defaults]' 'issue_type = "Task"' 'epic_link = "OPS-12"')
run_expect "$H" '
expect "Summary" { send "No epic\r" }
expect "> " { send ".\r" }
expect "Search epics" { send "\r" }
expect "open epics" { sleep 0.5; send "Skip" }
sleep 0.5
send "\r"
expect "Create?" { send "\r" }'
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

# --- scripted create: flags, stdin description, field by name, key-only stdout
H=$(new_home '' '[issue_defaults]' 'epic_link = "OPS-40"')
: > "$STUB_LOG"
set +e
OUT=$(printf 'Steps\n\nto reproduce\n' | HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=t "$BIN" create -s "Fix login" -t bug -d - -F "Story Points=3" -y 2>/dev/null)
EXIT=$?
set -e
if [ "$EXIT" = 0 ] && [ "$OUT" = "OPS-99" ] && python3 - "$STUB_LOG" <<'PY'
import json, sys
f = json.loads(open(sys.argv[1]).read())["fields"]
assert f["issuetype"]["name"] == "Bug", f
assert f["description"] == "Steps\n\nto reproduce", f
assert f["customfield_10016"] == "3", f
assert f["parent"]["key"] == "OPS-40", f
PY
then
  pass "scripted create"
else
  fail "scripted create" "exit=$EXIT out=$OUT $(cat "$STUB_LOG")"
fi

# --- scripted create: missing type is an error, nothing created
H=$(new_home)
: > "$STUB_LOG"
set +e
ERR=$(HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=t "$BIN" create -s x -y 2>&1 >/dev/null)
EXIT=$?
set -e
if [ "$EXIT" = 1 ] && [[ "$ERR" == *"issue type required: pass -t or set issue_defaults.issue_type"* ]] && [ ! -s "$STUB_LOG" ]; then
  pass "scripted create: missing type"
else
  fail "scripted create: missing type" "exit=$EXIT err=$ERR"
fi

# --- scripted create: unknown field suggests close names
set +e
ERR=$(HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=t "$BIN" create -s x -t Task -F "Story Pints=1" -y 2>&1 >/dev/null)
EXIT=$?
set -e
if [ "$EXIT" = 1 ] && [[ "$ERR" == *"Story Points (customfield_10016)"* ]]; then
  pass "scripted create: unknown field"
else
  fail "scripted create: unknown field" "exit=$EXIT err=$ERR"
fi

# --- interactive: paragraphs, field names in review, edit summary, editor
H=$(new_home '' '[issue_defaults]' 'issue_type = "Task"' '[issue_defaults.custom_fields]' "customfield_15838 = '{\"value\": \"Ops\"}'")
printf '#!/bin/sh\nprintf "Edited in editor\\n\\nSecond paragraph\\n" > "$1"\n' > "$WORK/editor.sh"
chmod +x "$WORK/editor.sh"
run_expect "$H" '
expect "Summary" { send "Typo summry\r" }
expect "> " { send "First\r" }
expect "> " { send "\r" }
expect "> " { send "Second\r" }
expect "> " { send ".\r" }
expect "Select epic" { sleep 0.5; send "(None)" }
sleep 0.5
send "\r"
expect "Work Allocation:" {}
expect "Create?" { send "e\r" }
expect "Edit which field" { sleep 0.5; send "Summary" }
sleep 0.5
send "\r"
expect "Summary" { send "Fixed summary\r" }
expect "Create?" { send "d\r" }
expect "Create?" { send "\r" }' "create"
if python3 - "$STUB_LOG" <<'PY'
import json, sys
f = json.loads(open(sys.argv[1]).read())["fields"]
assert f["summary"] == "Fixed summary", f
assert f["description"] == "Edited in editor\n\nSecond paragraph", f
assert f["customfield_15838"] == {"value": "Ops"}, f
assert "parent" not in f, f
PY
then
  pass "interactive: edit and editor"
else
  fail "interactive: edit and editor" "$(cat "$STUB_LOG")"
fi

exit $FAILED
