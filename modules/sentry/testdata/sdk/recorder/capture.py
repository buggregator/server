"""Records every HTTP request (method, path, headers, raw body) as JSON. Usage: capture.py <outdir> <port>"""
import base64, json, os, sys, threading
from http.server import ThreadingHTTPServer, BaseHTTPRequestHandler
out, port = sys.argv[1], int(sys.argv[2])
lock = threading.Lock(); n = [0]
class H(BaseHTTPRequestHandler):
    def cors(self):
        o = self.headers.get("Origin")
        if o:
            self.send_header("Access-Control-Allow-Origin", o)
            self.send_header("Access-Control-Allow-Headers", self.headers.get("Access-Control-Request-Headers", "*"))
            self.send_header("Access-Control-Allow-Methods", "POST, OPTIONS")
    def save(self, body):
        sdk = self.headers.get("X-Rig-Sdk") or "unknown"
        with lock:
            n[0] += 1; i = n[0]
        d = os.path.join(out, "all"); os.makedirs(d, exist_ok=True)
        rec = {"method": self.command, "path": self.path, "headers": dict(self.headers.items()), "body_b64": base64.b64encode(body).decode()}
        with open(os.path.join(d, f"{i:04d}.json"), "w") as f: json.dump(rec, f, indent=1)
    def do_OPTIONS(self):
        self.save(b"")
        self.send_response(204); self.cors(); self.end_headers()
    def do_POST(self):
        if self.headers.get("Transfer-Encoding", "").lower() == "chunked":
            body = b""
            while True:
                size = int(self.rfile.readline().split(b";")[0].strip(), 16)
                if size == 0:
                    while self.rfile.readline() not in (b"\r\n", b"\n", b""): pass
                    break
                body += self.rfile.read(size); self.rfile.readline()
        else:
            body = self.rfile.read(int(self.headers.get("Content-Length") or 0))
        self.save(body)
        self.send_response(200); self.cors(); self.send_header("Content-Type", "application/json"); self.end_headers(); self.wfile.write(b"{}")
    def do_GET(self):
        p = os.path.join(out, "static", self.path.lstrip("/").split("?")[0])
        if os.path.isfile(p):
            data = open(p, "rb").read()
            self.send_response(200); self.send_header("Content-Type", "text/html" if p.endswith(".html") else "application/javascript"); self.end_headers(); self.wfile.write(data)
        else:
            self.send_response(404); self.end_headers()
    def log_message(self, *a): pass
ThreadingHTTPServer(("127.0.0.1", port), H).serve_forever()
