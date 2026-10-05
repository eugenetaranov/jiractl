"""Minimal Jira stub for end-to-end tests.

Every create, transition, assign and comment request is appended to $STUB_LOG
as one JSON line: {"method", "path", "body"}. Searches append {"jql": ...}.
The token "bad" is rejected with 401.
"""
import base64, http.server, json, os, sys, urllib.parse

EPICS = {
    "OPS-40": {"key": "OPS-40", "fields": {"summary": "DevOps k8s cluster upgrade", "issuetype": {"name": "Epic"}}},
    "OPS-41": {"key": "OPS-41", "fields": {"summary": "Billing revamp", "issuetype": {"name": "Epic"}}},
}
ADF = {"type": "doc", "version": 1, "content": [
    {"type": "paragraph", "content": [{"type": "text", "text": "Login fails on Safari."}]}]}
TASKS = [
    {"key": "OPS-1", "fields": {"summary": "Fix login", "status": {"name": "To Do"}, "issuetype": {"name": "Bug"},
                                "assignee": {"displayName": "Ann"}, "description": ADF}},
    {"key": "OPS-2", "fields": {"summary": "Write docs", "status": {"name": "In Progress"}, "issuetype": {"name": "Task"}}},
]


def log(entry):
    with open(os.environ["STUB_LOG"], "a") as f:
        f.write(json.dumps(entry) + "\n")


class H(http.server.BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def send(self, code, obj=None):
        body = b"" if obj is None else json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def token(self):
        auth = self.headers.get("Authorization", "")
        if not auth.startswith("Basic "):
            return None
        return base64.b64decode(auth[6:]).decode().split(":", 1)[1]

    def do_GET(self):
        u = urllib.parse.urlparse(self.path)
        q = urllib.parse.parse_qs(u.query)
        if u.path == "/rest/api/2/serverInfo":
            return self.send(200, {"version": "1001.0.0", "deploymentType": "Cloud", "baseUrl": "http://stub"})
        if self.token() == "bad":
            return self.send(401)
        if u.path == "/rest/api/2/myself":
            return self.send(200, {"accountId": "me", "displayName": "Me"})
        if u.path == "/rest/api/2/project":
            return self.send(200, [{"key": "OPS", "name": "Operations"}, {"key": "WEB", "name": "Website"}])
        if u.path in ("/rest/api/2/project/OPS", "/rest/api/2/project/WEB"):
            return self.send(200, {"key": "OPS", "issueTypes": [{"name": "Task"}, {"name": "Bug"}]})
        if u.path == "/rest/api/2/field":
            return self.send(200, [
                {"id": "summary", "name": "Summary", "custom": False},
                {"id": "customfield_10016", "name": "Story Points", "custom": True},
                {"id": "customfield_15838", "name": "Work Allocation", "custom": True},
            ])
        if u.path == "/rest/api/2/mypermissions":
            names = q.get("permissions", [""])[0].split(",")
            return self.send(200, {"permissions": {n: {"havePermission": True} for n in names}})
        if u.path == "/rest/api/2/project/OPS/components":
            return self.send(200, [{"name": "Backend"}])
        if u.path == "/rest/api/2/issue/createmeta/OPS/issuetypes":
            return self.send(200, {"issueTypes": [{"id": "1", "name": "Task"}, {"id": "2", "name": "Bug"}]})
        if u.path.startswith("/rest/api/2/issue/createmeta/OPS/issuetypes/"):
            return self.send(200, {"fields": [{"fieldId": "summary", "name": "Summary", "required": True}]})
        if u.path.endswith("/transitions"):
            return self.send(200, {"transitions": [{"id": "21", "name": "Start", "to": {"name": "In Progress"}}]})
        if u.path.startswith("/rest/api/2/issue/"):
            key = u.path.rsplit("/", 1)[1]
            if key in EPICS:
                return self.send(200, EPICS[key])
            return self.send(404, {"errorMessages": ["Issue does not exist or you do not have permission to see it."]})
        if u.path == "/rest/api/3/search/jql":
            jql = q.get("jql", [""])[0]
            log({"jql": jql})
            low = jql.lower()
            if "issuetype = epic" in low:
                if "devops" in low or "k8s" in low:
                    return self.send(200, {"issues": [EPICS["OPS-40"]]})
                return self.send(200, {"issues": list(EPICS.values())})
            if "bad" in low:
                return self.send(400, {"errorMessages": ["Error in the JQL Query: 'bad' is not a field."]})
            return self.send(200, {"issues": TASKS})
        self.send(404, {"errorMessages": ["no stub for " + u.path]})

    def write(self, method):
        n = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(n) or b"{}")
        log({"method": method, "path": self.path, "body": body})
        if self.path == "/rest/api/2/issue":
            return self.send(201, {"key": "OPS-99"})
        if self.path.endswith("/comment"):
            return self.send(201, {"id": "1"})
        return self.send(204)

    def do_POST(self):
        if self.path.startswith("/rest/api/3/jql/parse"):
            n = int(self.headers.get("Content-Length", 0))
            queries = json.loads(self.rfile.read(n))["queries"]
            return self.send(200, {"queries": [
                {"query": q, "errors": ["Error in the JQL Query: 'bad' is not a field."] if "bad" in q else []}
                for q in queries]})
        self.write("POST")

    def do_PUT(self):
        self.write("PUT")


http.server.HTTPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()
