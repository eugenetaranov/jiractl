"""Minimal Jira stub for end-to-end tests. Records create bodies in $STUB_LOG."""
import http.server, json, os, sys, urllib.parse

EPICS = {
    "OPS-40": {"key": "OPS-40", "fields": {"summary": "DevOps k8s cluster upgrade", "issuetype": {"name": "Epic"}}},
    "OPS-41": {"key": "OPS-41", "fields": {"summary": "Billing revamp", "issuetype": {"name": "Epic"}}},
}

class H(http.server.BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def send(self, code, obj):
        body = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        u = urllib.parse.urlparse(self.path)
        q = urllib.parse.parse_qs(u.query)
        if u.path == "/rest/api/2/myself":
            return self.send(200, {"accountId": "me", "displayName": "Me"})
        if u.path == "/rest/api/2/project/OPS":
            return self.send(200, {"key": "OPS", "issueTypes": [{"name": "Task"}, {"name": "Bug"}]})
        if u.path.startswith("/rest/api/2/issue/"):
            key = u.path.rsplit("/", 1)[1]
            if key in EPICS:
                return self.send(200, EPICS[key])
            return self.send(404, {"errorMessages": ["Issue does not exist or you do not have permission to see it."]})
        if u.path == "/rest/api/2/field":
            return self.send(200, [
                {"id": "summary", "name": "Summary", "custom": False},
                {"id": "customfield_10016", "name": "Story Points", "custom": True},
                {"id": "customfield_15838", "name": "Work Allocation", "custom": True},
            ])
        if u.path == "/rest/api/3/search/jql":
            jql = q.get("jql", [""])[0].lower()
            if "devops" in jql or "k8s" in jql:
                return self.send(200, {"issues": [EPICS["OPS-40"]]})
            return self.send(200, {"issues": list(EPICS.values())})
        self.send(404, {"errorMessages": ["no stub for " + u.path]})

    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(n) or b"{}")
        if self.path == "/rest/api/2/issue":
            with open(os.environ["STUB_LOG"], "a") as f:
                f.write(json.dumps(body) + "\n")
            return self.send(201, {"key": "OPS-99"})
        self.send(404, {})

http.server.HTTPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()
