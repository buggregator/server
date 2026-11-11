#!/usr/bin/env bash
# Starts Buggregator on a fresh database, fills it with demo data and keeps it
# running until Ctrl-C. The database lives in a temp dir and is removed on exit.
#
#   scripts/demo/run.sh [path/to/buggregator]
#
# Binary: first argument, else $BUGGREGATOR_BIN, else built from this checkout.
# Ports: HTTP_PORT (8000), SMTP_PORT (1025), MONOLOG_PORT (9913),
#        VARDUMPER_PORT (9912), PROXY_PORT (8080).
# SKIP_VARDUMPER=1 skips the PHP dump() demo (it needs Docker).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HTTP_PORT="${HTTP_PORT:-8000}"
SMTP_PORT="${SMTP_PORT:-1025}"
MONOLOG_PORT="${MONOLOG_PORT:-9913}"
VARDUMPER_PORT="${VARDUMPER_PORT:-9912}"
PROXY_PORT="${PROXY_PORT:-8080}"

WORK="$(mktemp -d -t buggregator-demo.XXXXXX)"
SERVER_PID=""
cleanup() {
  if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  rm -rf "$WORK"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

BIN="${1:-${BUGGREGATOR_BIN:-}}"
if [[ -z "$BIN" ]]; then
  echo "building server..."
  (cd "$ROOT" && go build -o "$WORK/buggregator" ./cmd/buggregator)
  BIN="$WORK/buggregator"
fi

URL="http://127.0.0.1:$HTTP_PORT"
if curl -s -o /dev/null "$URL/" 2>/dev/null; then
  echo "something already listens on $URL: stop it or set HTTP_PORT" >&2
  exit 1
fi

echo "starting $BIN (data in $WORK)"
# VarDumper listens on all interfaces so the Docker container can reach it.
PROXY_ADDR="127.0.0.1:$PROXY_PORT" "$BIN" \
  -db "$WORK/demo.db" \
  -http-addr "127.0.0.1:$HTTP_PORT" \
  -smtp-addr "127.0.0.1:$SMTP_PORT" \
  -monolog-addr "127.0.0.1:$MONOLOG_PORT" \
  -vardumper-addr "0.0.0.0:$VARDUMPER_PORT" \
  >"$WORK/server.log" 2>&1 &
SERVER_PID=$!

for _ in $(seq 1 50); do
  if curl -sf "$URL/api/version" >/dev/null 2>&1; then break; fi
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    echo "server exited:" >&2
    cat "$WORK/server.log" >&2
    exit 1
  fi
  sleep 0.2
done
if ! kill -0 "$SERVER_PID" 2>/dev/null || ! curl -sf "$URL/api/version" >/dev/null; then
  echo "server did not start on $URL:" >&2
  cat "$WORK/server.log" >&2
  exit 1
fi

(cd "$ROOT" && go run ./scripts/demo/seed \
  -http "$URL" -smtp "127.0.0.1:$SMTP_PORT" -monolog "127.0.0.1:$MONOLOG_PORT")

if [[ "${SKIP_VARDUMPER:-}" != "1" ]]; then
  if command -v docker >/dev/null 2>&1; then
    echo "seeding var-dump (Docker)..."
    if docker build -q -t buggregator-demo-dumps "$ROOT/scripts/demo/php" >/dev/null; then
      docker run --rm --add-host=host.docker.internal:host-gateway \
        -e "VAR_DUMPER_SERVER=host.docker.internal:$VARDUMPER_PORT" buggregator-demo-dumps ||
        echo "var-dump demo failed, the rest of the data is in place" >&2
    else
      echo "could not build the var-dump image, skipped" >&2
    fi
  else
    echo "docker not found: var-dump demo skipped (see scripts/demo/README.md)"
  fi
fi

echo
echo "Buggregator demo is running: $URL"
echo "Server log: $WORK/server.log. Press Ctrl-C to stop (the demo data is removed)."
wait "$SERVER_PID"
