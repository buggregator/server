"""Scrubs recorded requests and writes them as fixtures: export.py <srcdir> <dstdir>"""
import base64, gzip, json, os, sys, brotli
src, dst = sys.argv[1], sys.argv[2]
SCRATCH = os.environ["SCRATCH"]
def _local():
    """Machine details to hide: host, user, home, time zone, CPU, kernel."""
    import getpass, platform, socket
    cpu = next((l.split(":", 1)[1].strip() for l in open("/proc/cpuinfo") if l.startswith("model name")), "")
    tz = os.path.realpath("/etc/localtime").split("zoneinfo/")[-1]
    out = [(os.path.expanduser("~") + "/go/pkg/mod", "/root/go/pkg/mod"), ("/snap/go/current", "/usr/local/go"),
           (socket.gethostname(), "host-01"), (tz, "UTC"), (platform.release(), "6.8.0-generic")]
    if cpu: out.append((cpu, "Test CPU"))
    out.append((getpass.getuser(), "dev"))
    return out
REPL = [(SCRATCH + "/rig/scripts", "/app/src"), (SCRATCH + "/sdk-go", "/app"), (SCRATCH + "/sdk-node", "/app"),
        (SCRATCH + "/sdk-py", "/app/.venv"), (SCRATCH + "/rig/out/static", "/app/public"), (SCRATCH, "/app")] + _local()
KEEP = {"content-type", "content-encoding", "x-sentry-auth", "origin", "user-agent", "transfer-encoding"}
def scrub(b):
    s = b.decode("utf-8")
    for a, c in REPL: s = s.replace(a, c)
    return s.encode("utf-8")
def envelope(raw):
    """Re-serializes an envelope, scrubbing each payload and fixing explicit lengths."""
    out = []; head, _, rest = raw.partition(b"\n"); out.append(scrub(head))
    while rest:
        ih, _, rest = rest.partition(b"\n")
        if not ih.strip(): continue
        hdr = json.loads(ih)
        if "length" in hdr:
            payload, rest = rest[:hdr["length"]], rest[hdr["length"]:]
            if rest.startswith(b"\n"): rest = rest[1:]
            payload = scrub(payload); hdr["length"] = len(payload)
            out.append(json.dumps(hdr, separators=(",", ":")).encode())
        else:
            payload, _, rest = rest.partition(b"\n"); payload = scrub(payload); out.append(scrub(ih))
        out.append(payload)
    return b"\n".join(out) + b"\n"
os.makedirs(dst, exist_ok=True); n = 0
for f in sorted(os.listdir(src)):
    d = json.load(open(os.path.join(src, f)))
    if d["method"] != "POST": continue
    h = {k.lower(): v for k, v in d["headers"].items()}
    b = base64.b64decode(d["body_b64"]); enc = h.get("content-encoding", "")
    raw = brotli.decompress(b) if enc == "br" else gzip.decompress(b) if enc == "gzip" else b
    raw = envelope(raw)
    kinds = []
    for ln in raw.split(b"\n")[1:]:
        try:
            j = json.loads(ln)
            if isinstance(j, dict) and set(j) & {"length", "content_type", "item_count"} or (isinstance(j, dict) and list(j) == ["type"]): kinds.append(j["type"])
        except Exception: pass
    body = brotli.compress(raw) if enc == "br" else gzip.compress(raw, mtime=0) if enc == "gzip" else raw
    n += 1
    name = f"{n:02d}-{'-'.join(kinds) or 'unknown'}.json"
    rec = {"method": "POST", "path": scrub(d["path"].encode()).decode(), "headers": {k: v for k, v in h.items() if k in KEEP and k != "transfer-encoding"},
           "body": raw.decode("utf-8") if not enc else None, "body_b64": base64.b64encode(body).decode() if enc else None}
    rec = {k: v for k, v in rec.items() if v is not None}
    json.dump(rec, open(os.path.join(dst, name), "w"), indent=1, ensure_ascii=False); open(os.path.join(dst, name), "a").write("\n")
    print(name, len(raw))
