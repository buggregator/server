#!/bin/sh
# Buggregator installer for Linux and macOS.
#
#   curl -fsSL https://raw.githubusercontent.com/buggregator/server/master/scripts/install.sh | sh
#
# Downloads the binary for this OS/CPU from a GitHub release, checks it against
# the release's checksums.txt, and puts it in ~/.local/bin (no sudo).
#
# Environment:
#   BUGGREGATOR_VERSION      release tag, e.g. v2.6.0 (default: latest)
#   BUGGREGATOR_INSTALL_DIR  where to put the binary (default: ~/.local/bin)
#   BUGGREGATOR_BASE_URL     download base URL instead of GitHub (mirrors, tests)
#   BUGGREGATOR_NO_VERIFY=1  skip the checksum (only for releases without checksums.txt)
set -eu

REPO="buggregator/server"
VERSION="${BUGGREGATOR_VERSION:-latest}"
INSTALL_DIR="${BUGGREGATOR_INSTALL_DIR:-$HOME/.local/bin}"

fail() { echo "buggregator install: $*" >&2; exit 1; }

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) fail "unsupported OS $(uname -s); on Windows use install.ps1" ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) fail "unsupported CPU $(uname -m)" ;;
esac
asset="buggregator-$os-$arch"

if [ -n "${BUGGREGATOR_BASE_URL:-}" ]; then
  base="$BUGGREGATOR_BASE_URL"
elif [ "$VERSION" = latest ]; then
  base="https://github.com/$REPO/releases/latest/download"
else
  base="https://github.com/$REPO/releases/download/$VERSION"
fi

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -q -O "$2" "$1"; }
else
  fail "needs curl or wget"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

echo "Downloading $asset ($VERSION)..."
fetch "$base/$asset" "$tmp/$asset" || fail "download failed: $base/$asset"

if [ "${BUGGREGATOR_NO_VERIFY:-}" = 1 ]; then
  echo "Checksum NOT verified (BUGGREGATOR_NO_VERIFY=1)."
else
  fetch "$base/checksums.txt" "$tmp/checksums.txt" ||
    fail "no checksums.txt in this release; set BUGGREGATOR_NO_VERIFY=1 to install without a check"
  want="$(awk -v a="$asset" '$2 == a || $2 == "*" a { print $1 }' "$tmp/checksums.txt")"
  [ -n "$want" ] || fail "$asset is not listed in checksums.txt"
  if command -v sha256sum >/dev/null 2>&1; then
    got="$(sha256sum "$tmp/$asset" | awk '{ print $1 }')"
  elif command -v shasum >/dev/null 2>&1; then
    got="$(shasum -a 256 "$tmp/$asset" | awk '{ print $1 }')"
  else
    fail "needs sha256sum or shasum to check the download"
  fi
  [ "$got" = "$want" ] || fail "checksum mismatch for $asset (got $got, want $want)"
  echo "Checksum OK."
fi

mkdir -p "$INSTALL_DIR"
chmod 755 "$tmp/$asset"
mv "$tmp/$asset" "$INSTALL_DIR/buggregator"
bin="$INSTALL_DIR/buggregator"

echo
echo "Installed $bin"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) cmd=buggregator ;;
  *)
    cmd="$bin"
    echo "Note: $INSTALL_DIR is not in your PATH. Add it, or run $bin."
    ;;
esac
cat <<EOF

Start it (web UI at http://127.0.0.1:8000):
  $cmd

Only on this machine (no other hosts on the network):
  HTTP_ADDR=127.0.0.1:8000 SMTP_ADDR=127.0.0.1:1025 MONOLOG_ADDR=127.0.0.1:9913 VAR_DUMPER_ADDR=127.0.0.1:9912 PROXY_ADDR=127.0.0.1:8080 $cmd

With the MCP server for AI agents, then connect Claude Code:
  MCP_ENABLED=true $cmd
  claude mcp add buggregator -- $cmd mcp
EOF
