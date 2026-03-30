# Debug everything. Install nothing.

<a href="https://discord.gg/vDsCD3EKUB"><img src="https://img.shields.io/badge/discord-chat-magenta.svg"></a>
[![Twitter](https://img.shields.io/badge/twitter-Follow-blue)](https://twitter.com/buggregator)
[![Support me on Patreon](https://img.shields.io/endpoint.svg?url=https%3A%2F%2Fshieldsio-patreon.vercel.app%2Fapi%3Fusername%3Dbutschster%26type%3Dpatrons&style=flat)](https://patreon.com/butschster)
[![CI](https://github.com/buggregator/server/actions/workflows/ci.yml/badge.svg)](https://github.com/buggregator/server/actions/workflows/ci.yml)
[![Downloads](https://img.shields.io/docker/pulls/butschster/buggregator.svg)](https://hub.docker.com/repository/docker/butschster/buggregator)

<a href="https://www.producthunt.com/posts/buggregator?embed=true&utm_source=badge-featured&utm_medium=badge&utm_souce=badge-buggregator" target="_blank"><img src="https://api.producthunt.com/widgets/embed-image/v1/featured.svg?post_id=478631&theme=light" alt="Buggregator - The&#0032;Ultimate&#0032;Debugging&#0032;Server&#0032;for&#0032;PHP | Product Hunt" style="width: 250px; height: 54px;" width="250" height="54" /></a>

**One binary. Exceptions, dumps, emails, profiling, logs — all in one real-time UI. Works with the SDKs you already have. No cloud account. No code changes. No runtime dependencies.**

Buggregator v2.0 is a complete rewrite in Go — a single self-contained binary with an embedded SQLite database, web UI, and WebSocket server. No PHP runtime, no RoadRunner, no Centrifugo, no external database.

| | |
|---|---|
| **Server** | `v2.0.0` |
| **Frontend** | `v1.30.0` |

> Looking for the PHP version? See the [1.x branch](https://github.com/buggregator/server/tree/1.x).

Watch our introduction video on [YouTube](https://www.youtube.com/watch?v=yKWbuw8xN_c)

## Quick Start

```bash
docker run --pull always \
  -p 127.0.0.1:8000:8000 \
  -p 127.0.0.1:1025:1025 \
  -p 127.0.0.1:9912:9912 \
  -p 127.0.0.1:9913:9913 \
  ghcr.io/buggregator/server:latest
```

Open http://127.0.0.1:8000 and start debugging. That's it.

### Standalone Binary

```bash
# Download the latest release
curl -sL https://github.com/buggregator/server/releases/latest/download/buggregator-linux-amd64 -o buggregator
chmod +x buggregator
./buggregator
```

> No Docker? No binary? Use [Buggregator Trap](https://docs.buggregator.dev/trap/what-is-trap.html) — a lightweight PHP CLI alternative.

## Key Features

### [Xhprof Profiler](https://docs.buggregator.dev/config/xhprof.html)

Watch our intro video about profiler on [YouTube](https://www.youtube.com/watch?v=2QbgjIVnz78&pp=ygULYnVnZ3JlZ2F0b3I%3D).

![xhprof](https://github.com/buggregator/server/assets/773481/d69e1158-599d-4546-96a9-40a42cb060f4)

### [Symfony VarDumper Server](https://docs.buggregator.dev/config/var-dumper.html)

![var-dumper](https://github.com/buggregator/server/assets/773481/b77fa867-0a8e-431a-9126-f69959dc18f4)

### [Spatie Ray Debug Tool](https://docs.buggregator.dev/config/ray.html)

![ray](https://github.com/buggregator/server/assets/773481/168b27f7-75b1-4837-b0a1-37146d5b8b52)

### [Fake SMTP Server](https://docs.buggregator.dev/config/smtp.html)

![smtp](https://github.com/buggregator/server/assets/773481/8dd60ddf-c8d8-4a26-a8c0-b05052414a5f)

### [Sentry Compatibility](https://docs.buggregator.dev/config/sentry.html)

![sentry](https://github.com/buggregator/server/assets/773481/e979fda5-54c8-42cc-8224-a1c5d828569a)

### [Monolog Server](https://docs.buggregator.dev/config/monolog.html)

![monolog](https://github.com/buggregator/server/assets/773481/21919110-fd4d-490d-a78e-41242d329885)

### [Inspector Compatibility](https://docs.buggregator.dev/config/inspector.html)

![inspector](https://github.com/buggregator/server/assets/773481/ab002ecf-e1dc-4433-90d4-0e42ff8c0ab3)

### [HTTP Requests Dump Server](https://docs.buggregator.dev/config/http-dumps.html)

![http dumps](https://github.com/buggregator/server/assets/773481/fc823390-b490-4bbb-a787-44471eca9fb6)

## Supported Modules

| Module | Type | Transport | Description |
|--------|------|-----------|-------------|
| Sentry | `sentry` | HTTP | Error tracking (gzip, envelope format) |
| Ray | `ray` | HTTP | Debug tool for PHP |
| VarDumper | `var-dump` | TCP :9912 | Symfony VarDumper (embedded PHP parser) |
| Inspector | `inspector` | HTTP | APM monitoring |
| Monolog | `monolog` | TCP :9913 | Logging (newline-delimited JSON) |
| SMTP | `smtp` | TCP :1025 | Email capture (RFC822, multipart, attachments) |
| SMS | `sms` | HTTP `/sms` | SMS gateway capture (41 providers) |
| HTTP Dump | `http-dump` | HTTP | Catch-all HTTP request capture |
| Profiler | `profiler` | HTTP | XHProf profiling (call graph, flame chart, top functions) |
| Webhooks | `webhooks` | — | HTTP POST notifications when events are received |

## Configuration

The server works with zero configuration. Optionally, create a `buggregator.yaml`:

```yaml
server:
  addr: ":8000"

database:
  driver: sqlite
  dsn: ":memory:"           # or "data.db" for persistence

# Prometheus metrics (optional)
metrics:
  enabled: true
  addr: ":9090"             # separate server, or empty to use main HTTP

tcp:
  smtp:
    addr: ":1025"
  monolog:
    addr: ":9913"
  var-dumper:
    addr: ":9912"

# Authentication (disabled by default)
# Supports: auth0, google, github, keycloak, gitlab, oidc (generic)
auth:
  enabled: true
  provider: auth0
  provider_url: https://your-tenant.us.auth0.com
  client_id: your-client-id
  client_secret: your-client-secret
  callback_url: http://localhost:8000/auth/sso/callback
  scopes: openid,email,profile
  jwt_secret: your-secret-for-signing-tokens

# MCP — AI assistant integration (disabled by default)
mcp:
  enabled: true
  transport: socket              # "socket" or "http"
  socket_path: /tmp/buggregator-mcp.sock

# Enable/disable modules (all enabled by default)
modules:
  sentry: true
  ray: true
  var-dump: true
  inspector: true
  monolog: true
  smtp: true
  sms: true
  http-dump: true
  profiler: true

# Webhooks — HTTP POST notifications on events
webhooks:
  - event: "*"
    url: https://slack.example.com/webhook
    headers:
      Authorization: "Bearer token"
    verify_ssl: false
    retry: true
  - event: sentry
    url: https://pagerduty.example.com/alert

# Pre-defined projects
projects:
  - key: my-app
    name: My Application
```

### Configuration Priority

1. **CLI flags** (`--http-addr :9000`)
2. **Environment variables** (`HTTP_ADDR=:9000`)
3. **Config file** (`buggregator.yaml`)
4. **Defaults**

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_ADDR` | `:8000` | HTTP listen address |
| `DATABASE_DSN` | `:memory:` | SQLite DSN |
| `SMTP_ADDR` | `:1025` | SMTP listen address |
| `MONOLOG_ADDR` | `:9913` | Monolog TCP address |
| `VAR_DUMPER_ADDR` | `:9912` | VarDumper TCP address |
| `CLIENT_SUPPORTED_EVENTS` | all | Comma-separated list of enabled modules |
| `METRICS_ENABLED` | `false` | Enable Prometheus metrics |
| `METRICS_ADDR` | — | Separate metrics server address |
| `AUTH_ENABLED` | `false` | Enable OAuth2/OIDC authentication |
| `AUTH_PROVIDER` | `oidc` | Provider: `auth0`, `google`, `github`, `keycloak`, `gitlab`, `oidc` |
| `AUTH_PROVIDER_URL` | — | OIDC issuer URL |
| `AUTH_CLIENT_ID` | — | OAuth2 client ID |
| `AUTH_CLIENT_SECRET` | — | OAuth2 client secret |
| `AUTH_CALLBACK_URL` | — | OAuth2 callback URL |
| `AUTH_SCOPES` | `openid,email,profile` | Comma-separated OAuth2 scopes |
| `AUTH_JWT_SECRET` | — | Secret for signing internal JWT tokens |
| `MCP_ENABLED` | `false` | Enable MCP server |
| `MCP_TRANSPORT` | `socket` | `socket` (local) or `http` (remote) |
| `MCP_SOCKET_PATH` | `/tmp/buggregator-mcp.sock` | Unix socket path |
| `MCP_ADDR` | `:8001` | HTTP listen address for MCP |
| `MCP_AUTH_TOKEN` | — | Bearer token for HTTP MCP transport |

## PHP Client Configuration

### Laravel / Symfony

```env
# .env
SENTRY_LARAVEL_DSN=http://sentry@localhost:8000/default
RAY_HOST=ray@localhost
RAY_PORT=8000
VAR_DUMPER_SERVER=localhost:9912
LOG_SOCKET_URL=localhost:9913
MAIL_HOST=localhost
MAIL_PORT=1025
INSPECTOR_URL=http://inspector@localhost:8000
INSPECTOR_API_KEY=test
PROFILER_ENDPOINT=http://profiler@localhost:8000
HTTP_DUMP_ENDPOINT=http://http-dump@localhost:8000
SMS_ENDPOINT=http://localhost:8000/sms
```

## MCP (Model Context Protocol)

Buggregator includes a built-in MCP server that lets AI assistants (Claude Code, Cursor, etc.) query and analyze debugging data directly.

```
AI Assistant --stdio--> buggregator mcp --unix socket--> buggregator (main process)
```

### Setup

**Claude Code** (`~/.claude.json` or project settings):
```json
{
  "mcpServers": {
    "buggregator": {
      "command": "./buggregator",
      "args": ["mcp"]
    }
  }
}
```

**Cursor** (`.cursor/mcp.json`):
```json
{
  "mcpServers": {
    "buggregator": {
      "command": "./buggregator",
      "args": ["mcp"]
    }
  }
}
```

### Available Tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `events_list` | List captured events | `type`, `project`, `limit` (default 20, max 100) |
| `event_get` | Get full event payload | `uuid` |
| `event_delete` | Delete an event | `uuid` |
| `profiler_summary` | Profile overview: totals, slowest function, memory hotspot | `uuid` |
| `profiler_top` | Top functions sorted by metric | `uuid`, `metric`, `limit` |
| `profiler_call_graph` | Filtered call graph with significant paths | `uuid`, `metric`, `percentage`, `threshold` |
| `sentry_event` | Structured error: exception chain, stack traces, tags | `uuid` |
| `vardump_get` | Variable value with HTML stripped | `uuid` |

## Building from Source

### Prerequisites

- Go 1.22+
- PHP 8.1+ with Composer (for VarDumper module)
- Make

```bash
make build          # Full build: downloads frontend, builds PHP parser, compiles binary
make run            # Build and run
make build-cross GOOS=darwin GOARCH=arm64   # Cross-compile
```

## Documentation

See the [documentation](https://docs.buggregator.dev/) for detailed installation and usage instructions.

## Contributing

We enthusiastically invite you to contribute to Buggregator Server! Whether you've uncovered a bug, have innovative feature suggestions, or wish to contribute in any other capacity, we warmly welcome your participation. Simply open an issue or submit a pull request on our GitHub repository to get started.

> Read more about how to contribute [here](https://docs.buggregator.dev/contributing.html).

## License

Buggregator is open-sourced software licensed under the MIT License.
