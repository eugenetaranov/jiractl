#!/usr/bin/env bash
# End-to-end tests: drive the real binary through a pty (expect) against a
# local Jira stub. Uses an in-memory keyring (-tags keyringmock) and a
# throwaway HOME, so nothing on the machine is touched.
#
# Usage: tests/e2e/run.sh            (requires expect and python3)
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
WORK=$(mktemp -d)
trap 'kill $STUB_PID $DC_PID 2>/dev/null || true; rm -rf "$WORK"' EXIT

PORT=${PORT:-18799}
BIN=$WORK/jiractl
go build -tags keyringmock -o "$BIN" "$ROOT/cmd/jiractl"

export STUB_LOG=$WORK/creates.log
python3 "$ROOT/tests/e2e/stub.py" "$PORT" &
STUB_PID=$!
DC_PORT=$((PORT + 1))
STUB_DEPLOYMENT=Server python3 "$ROOT/tests/e2e/stub.py" "$DC_PORT" &
DC_PID=$!
sleep 0.5

FAILED=0
pass() { echo "ok   $1"; }
fail() {
  echo "FAIL $1: $2"
  FAILED=1
  if [ -f "$WORK/out.log" ]; then
    echo "  --- last terminal output:"
    LC_ALL=C tr -d '\033' < "$WORK/out.log" | LC_ALL=C tr '\r' '\n' | grep -a -v '^\s*$' | tail -15 | sed 's/^/  | /'
  fi
}

# new_home writes a config and prints the HOME directory.
new_home() {
  local h
  h=$(mktemp -d "$WORK/home.XXXX")
  printf '%s\n' "server = \"http://127.0.0.1:$PORT\"" 'project = "OPS"' "$@" > "$h/.jiractl.toml"
  echo "$h"
}

# run_expect HOME SCRIPT: runs jiractl with the given expect body.
run_expect() {
  local home=$1 body=$2 args=${3-create}
  : > "$STUB_LOG"
  cat > "$WORK/s.exp" <<EXP
set timeout 10
set stty_init "rows 40 cols 120"
log_file -noappend $WORK/out.log
spawn env EDITOR=$WORK/editor.sh HOME=$home XDG_STATE_HOME=$home/state JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=t $BIN $args
$body
expect {
  eof {}
  timeout { exec kill -9 [exp_pid] }
}
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
expect "Ctrl+D to finish" { send "\004" }
expect "Search epics" { send "devops k8s\r" }
expect "1 epics match" { sleep 0.5; send "OPS-40" }
sleep 0.5
send "\r"
expect "default epic?" { send "y" }
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
expect "Ctrl+D to finish" { send "\004" }
expect "Search epics" { send "\r" }
expect "open epics" { sleep 0.5; send "Skip" }
sleep 0.5
send "\r"
expect "Create?" { send "\r" }'
if grep -q '"method": "POST"' "$STUB_LOG" && ! grep -q parent "$STUB_LOG" && grep -q 'epic_link = "OPS-12"' "$H/.jiractl.toml"; then
  pass "missing epic: skip"
else
  fail "missing epic: skip" "$(cat "$STUB_LOG")"
fi

# --- configure: rejected token is re-asked; three failures save nothing
H=$(new_home '# my comment')
cp "$H/.jiractl.toml" "$WORK/orig"
run_expect "$H" '
expect "Server URL" { send "\r" }
expect "Username" { send "u\r" }
expect "API Token" { send "bad\r" }
expect "Username" { send "u\r" }
expect "API Token" { send "bad\r" }
expect "Username" { send "u\r" }
expect "API Token" { send "bad\r" }' configure
if [ "$EXIT" = 1 ] && cmp -s "$WORK/orig" "$H/.jiractl.toml" && out_has "credentials rejected"; then
  pass "configure: rejected token saves nothing"
else
  fail "configure: rejected token saves nothing" "exit=$EXIT"
fi

# --- first run: offer setup, scheme added, project picked, starter queries, menu
H=$(mktemp -d "$WORK/home.XXXX")
run_expect "$H" '
expect "Run setup now?" { send "\r" }
expect "Server URL" { send "http://127.0.0.1:'"$PORT"'/\r" }
expect "Username" { send "u\r" }
expect "API Token" { send "good\r" }
expect "Select project" { sleep 0.5; send "OPS" }
sleep 0.5
send "\r"
expect "default issue type" { sleep 0.5; send "\033" }
expect "default epic" { sleep 0.5; send "\033" }
expect "Select action" { sleep 0.5; send "\033" }' ""
if [ "$EXIT" = 0 ] && python3 - "$H/.jiractl.toml" <<'PY'
import sys, tomllib
c = tomllib.load(open(sys.argv[1], "rb"))
assert c["project"] == "OPS", c
assert [q["name"] for q in c["queries"]] == ["mine", "recent", "unassigned"], c
PY
then
  pass "first run: setup then menu"
else
  fail "first run: setup then menu" "exit=$EXIT $(cat "$H/.jiractl.toml" 2>/dev/null)"
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
f = [e for e in map(json.loads, open(sys.argv[1])) if e.get("path") == "/rest/api/2/issue"][0]["body"]["fields"]
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
if [ "$EXIT" = 1 ] && [[ "$ERR" == *"issue type required: pass -t or set issue_defaults.issue_type"* ]] && ! grep -q '"method"' "$STUB_LOG"; then
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
expect "Ctrl+D to finish" { send "First\r\rSecond"; sleep 0.3; send "\004" }
expect "Select epic" { sleep 0.5; send "(None)" }
sleep 0.5
send "\r"
expect "Work Allocation:" {}
expect "Create?" { send "e" }
expect "Edit which field" { sleep 0.5; send "Summary" }
sleep 0.5
send "\r"
expect -ex "\[Typo summry\]" { send "Fixed summary\r" }
expect "Create?" { send "d" }
expect "Create?" { send "\r" }' "create"
if python3 - "$STUB_LOG" <<'PY'
import json, sys
f = [e for e in map(json.loads, open(sys.argv[1])) if e.get("path") == "/rest/api/2/issue"][0]["body"]["fields"]
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

# --- query: prefix match, JSON output, ADF description flattened
H=$(new_home '[[queries]]' 'name = "mine"' 'jql = "project = ${project} AND assignee = currentUser()"')
set +e
OUT=$(HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=t "$BIN" query MI -o json 2>/dev/null)
EXIT=$?
set -e
if [ "$EXIT" = 0 ] && python3 -c '
import json, sys
d = json.loads(sys.argv[1])
assert d[0] == {"key": "OPS-1", "summary": "Fix login", "status": "To Do", "assignee": "Ann"}, d
' "$OUT"; then
  pass "query: prefix + json"
else
  fail "query: prefix + json" "exit=$EXIT out=$OUT"
fi

# --- query: piped output is a table, no picker, banner on stderr
set +e
OUT=$(HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=t "$BIN" query mine 2>/dev/null | cat)
set -e
ORDER=$(printf '%s\n' "$OUT" | awk '{print $1}' | tr '\n' ' ')
if [ "$ORDER" = "OPS-2 OPS-1 OPS-3 " ] && [[ "$OUT" != *"Running query"* ]]; then
  pass "query: piped table, sorted by status then update"
else
  fail "query: piped table" "$OUT"
fi

# --- query: unknown name lists available; --jql expands ${project}; bad JQL explained
set +e
ERR=$(HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=t "$BIN" query foo 2>&1)
EXIT1=$?
: > "$STUB_LOG"
HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=t "$BIN" query --jql 'project = ${project} AND labels = urgent' -o keys >/dev/null 2>&1
ERR2=$(HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=t "$BIN" query --jql 'bad = 1' -o keys 2>&1)
set -e
if [ "$EXIT1" = 1 ] && [[ "$ERR" == *'no query "foo"; available: mine'* ]] && grep -q '"jql": "project = OPS AND labels = urgent"' "$STUB_LOG" && [[ "$ERR2" == *"'bad' is not a field"* ]]; then
  pass "query: errors and --jql"
else
  fail "query: errors and --jql" "$ERR | $ERR2"
fi

# --- query: browse, preview, transition, back to list, Esc exits 0
: > "$STUB_LOG"
run_expect "$H" '
expect "mine (3 found)" { sleep 0.5; send "OPS-1" }
expect "Login fails on Safari" {}
sleep 0.3
send "\r"
expect "Open in browser" { sleep 0.5; send "Transition" }
sleep 0.5
send "\r"
expect "In Progress" { sleep 0.5; send "\r" }
expect "Back" { sleep 0.5; send "\033" }
expect "mine (3 found)" { sleep 0.5; send "\033" }' "query mine"
if [ "$EXIT" = 0 ] && grep '"path": "/rest/api/2/issue/OPS-1/transitions"' "$STUB_LOG" | grep -q '"transition": {"id": "21"}'; then
  pass "query: browse and transition"
else
  fail "query: browse and transition" "exit=$EXIT $(cat "$STUB_LOG")"
fi

# --- menu: header shows a missing default epic; change it, then clear it
H=$(new_home '' '[issue_defaults]' '# team epic' 'epic_link = "OPS-12"')
run_expect "$H" '
expect "Default epic: OPS-12 (not found)" { sleep 0.5; send "Change" }
sleep 0.5
send "\r"
expect "Change default epic (Esc keeps OPS-12)" { sleep 0.5; send "OPS-41" }
sleep 0.5
send "\r"
expect "Default epic: OPS-41 Billing revamp" { sleep 0.5; send "\033" }' ""
if [ "$EXIT" = 0 ] && grep -q 'epic_link = "OPS-41"' "$H/.jiractl.toml" && grep -q '# team epic' "$H/.jiractl.toml"; then
  pass "menu: change default epic"
else
  fail "menu: change default epic" "exit=$EXIT $(cat "$H/.jiractl.toml")"
fi
run_expect "$H" '
expect "Select action" { sleep 0.5; send "Change" }
sleep 0.5
send "\r"
expect "OPS-41" { sleep 0.5; send "None" }
sleep 0.5
send "\r"
expect "Default epic: none" { sleep 0.5; send "\033" }' ""
if [ "$EXIT" = 0 ] && ! grep -q 'epic_link' "$H/.jiractl.toml"; then
  pass "menu: clear default epic"
else
  fail "menu: clear default epic" "exit=$EXIT $(cat "$H/.jiractl.toml")"
fi

# --- doctor: healthy setup exits 0; a broken query and a rejected token exit 1
H=$(new_home '[issue_defaults]' 'issue_type = "Task"' 'epic_link = "OPS-40"' 'component = "Backend"' '[[queries]]' 'name = "mine"' 'jql = "assignee = currentUser()"')
chmod 600 "$H/.jiractl.toml"
set +e
OUT=$(HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=t "$BIN" doctor 2>&1)
EXIT1=$?
printf '[[queries]]\nname = "broken"\njql = "bad = 1"\n' >> "$H/.jiractl.toml"
JSON=$(HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=t "$BIN" doctor --json 2>/dev/null)
EXIT2=$?
OUT3=$(HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=bad "$BIN" doctor 2>&1)
EXIT3=$?
set -e
if [ "$EXIT1" = 0 ] && [[ "$OUT" == *"0 failed"* ]] && [ "$EXIT2" = 1 ] && python3 -c '
import json, sys
r = {x["id"]: x for x in json.loads(sys.argv[1])}
assert r["queries"]["status"] == "fail" and "broken:" in r["queries"]["detail"], r["queries"]
' "$JSON" && [ "$EXIT3" = 1 ] && [[ "$OUT3" == *"credentials rejected"* ]] && [[ "$OUT3" == *"[skip]"* ]]; then
  pass "doctor"
else
  fail "doctor" "exit=$EXIT1/$EXIT2/$EXIT3 $OUT | $OUT3"
fi

# --- Data Center: detected on setup, PAT only, Epic Link field, v2 search
H=$(mktemp -d "$WORK/home.XXXX")
run_expect "$H" '
expect "Server URL" { send "http://127.0.0.1:'"$DC_PORT"'\r" }
expect "Found Server" {}
expect "Personal Access Token" { send "pat\r" }
expect "Select project" { sleep 0.5; send "OPS" }
sleep 0.5
send "\r"
expect "default issue type" { sleep 0.5; send "Task" }
sleep 0.5
send "\r"
expect "default epic" { sleep 0.5; send "OPS-40" }
sleep 0.5
send "\r"
expect "Checking setup" {}' configure
: > "$STUB_LOG"
set +e
KEY=$(HOME=$H JIRACTL_TEST_TOKEN=pat "$BIN" create -s "DC issue" -y 2>"$WORK/err")
EXIT2=$?
KEYS=$(HOME=$H JIRACTL_TEST_TOKEN=pat "$BIN" query mine -o keys 2>/dev/null)
set -e
if grep -q 'deployment = "server"' "$H/.jiractl.toml" && [ "$EXIT2" = 0 ] && [ "$KEY" = OPS-99 ] &&
   grep '"path": "/rest/api/2/issue"' "$STUB_LOG" | grep -q '"customfield_10014": "OPS-40"' &&
   [ "$(echo $KEYS)" = "OPS-1 OPS-2 OPS-3" ]; then
  pass "data center: setup, create, query"
else
  fail "data center: setup, create, query" "exit=$EXIT2 key=$KEY keys=$KEYS $(cat "$WORK/err") $(cat "$H/.jiractl.toml")"
fi

# --- rejected token: menu warns up front; a failing action explains itself and waits
H=$(new_home '[[queries]]' 'name = "mine"' 'jql = "assignee = currentUser()"')
cat > "$WORK/s.exp" <<EXP
set timeout 10
set stty_init "rows 40 cols 120"
log_file -noappend $WORK/out.log
spawn env HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=bad $BIN
expect "Jira rejected your API token" { sleep 0.5; send "Run" }
sleep 0.5
send "\r"
expect "Select query" { sleep 0.5; send "\r" }
expect "Press Enter to return to the menu" { send "\r" }
expect "Run query failed" { sleep 0.5; send "\033" }
expect eof
EXP
set +e
expect "$WORK/s.exp" >/dev/null 2>&1
EXIT=$?
set -e
if [ "$EXIT" = 0 ] && out_has "rejected your credentials \(401\)" && out_has "Create a new API token"; then
  pass "menu: rejected token explained"
else
  fail "menu: rejected token explained" "exit=$EXIT"
fi

# --- command line: error plus hint
set +e
ERR=$(HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=bad "$BIN" query mine -o keys 2>&1 >/dev/null)
set -e
if [[ "$ERR" == *"rejected your credentials (401)"* ]] && [[ "$ERR" == *"→ Create a new API token"* ]]; then
  pass "cli: rejected token explained"
else
  fail "cli: rejected token explained" "$ERR"
fi

# --- expired token served as anonymous: still reported as rejected, not "no issues"
set +e
ERR=$(HOME=$H JIRACTL_TEST_USERNAME=u JIRACTL_TEST_TOKEN=expired "$BIN" query mine -o keys 2>&1 >/dev/null)
EXIT=$?
set -e
if [ "$EXIT" = 1 ] && [[ "$ERR" == *"rejected your credentials (401)"* ]]; then
  pass "cli: silently rejected token detected"
else
  fail "cli: silently rejected token detected" "exit=$EXIT $ERR"
fi

exit $FAILED
