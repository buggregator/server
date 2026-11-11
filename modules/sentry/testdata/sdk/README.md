# Requests recorded from real Sentry SDKs

Each `<sdk>/NN-<item types>.json` is one HTTP request an SDK sent: method,
path, the headers that matter, and the body as sent (`body`, or `body_b64`
when compressed). `sdk_fixtures_test.go` replays them through the handler.

| Dir | SDK | What is special |
|---|---|---|
| `node` | `@sentry/node` 11.11.0 (Node 22) | chunked bodies; traces as streamed `span` items (`is_segment` root), no `transaction` item |
| `python` | `sentry-python` 2.74.0 (Python 3.14) | `Content-Encoding: br` (sent when the `brotli` package is installed) |
| `go` | `sentry-go` 0.50.0 | `"exception": [...]` (array, not `{"values": [...]}`) |
| `browser` | `@sentry/browser` 11.11.0 in Chromium | cross-origin `text/plain` POST (no CORS preflight); streamed spans; pageload |

Each script sends: an exception with a breadcrumb, user and tag; a warning
message; a transaction `GET /orders` with a DB span and an HTTP client span
(browser: a pageload and a `checkout` span); two logs; sessions where the SDK
sends them.

## Recording again

`recorder/` holds the scripts. `capture.py <outdir> 18490` records every
request; each SDK script sends to `http://publickey@127.0.0.1:18490/42`:

```sh
python3 recorder/capture.py out 18490 &
NODE_PATH=<dir with @sentry/node> node recorder/node.js
<venv with sentry-sdk and brotli>/bin/python recorder/py.py
(cd <go module with sentry-go> && cp recorder/go-main.go.txt main.go && go run .)
esbuild recorder/browser-entry.js --bundle --format=iife --outfile=out/static/app.js
NODE_PATH=<dir with playwright-core> node recorder/browser-run.js   # serves out/static/index.html
```

Move `out/all` to `out/<sdk>` after each SDK, then
`SCRATCH=<recording dir> python recorder/export.py out/<sdk> <sdk>`: it
replaces local paths, host name, time zone and CPU with neutral values and
fixes envelope item lengths.
