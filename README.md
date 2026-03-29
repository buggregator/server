# Go-Buggregator

A lightweight, standalone debugging server for PHP applications — packed into a single Go binary.

Drop-in replacement for [Buggregator](https://github.com/buggregator/server) that requires no PHP runtime, no RoadRunner, no Centrifugo, and no external database.

## Features

- **Single binary** — download and run, zero dependencies
- **SQLite in-memory** by default — no database setup, no leftover files
- **Embedded frontend** — web UI served from the binary itself
- **Real-time updates** — built-in WebSocket with Centrifugo v5 protocol emulation
- **Modular architecture** — enable/disable modules via config
- **Cross-platform** — Linux, macOS (amd64, arm64)

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
| Webhooks | `webhooks` | — | Send HTTP POST notifications when events are received |

## Quick Start

```bash
# Download the latest release
curl -sL https://github.com/buggregator/go-server/releases/latest/download/buggregator-linux-amd64 -o buggregator
chmod +x buggregator

# Run with defaults (in-memory SQLite, all modules enabled)
./buggregator
```

Open http://localhost:8000 in your browser.

## Docker

```bash
# Run from Docker Hub
docker run -p 8000:8000 -p 1025:1025 -p 9912:9912 -p 9913:9913 ghcr.io/buggregator/go-server:latest

# Or with docker-compose (includes Laravel examples app)
docker compose up
```

## Building from Source

### Prerequisites

- Go 1.22+
- PHP 8.1+ with Composer (for VarDumper module build)
- Make

### Build

```bash
# Full build: downloads frontend, builds PHP parser, compiles Go binary
make build

# Build for all platforms
make release

# Build for a specific platform
make build-cross GOOS=darwin GOARCH=arm64
```

### Build Steps (what `make build` does)

1. **Frontend**: Downloads pre-built frontend from [buggregator/frontend](https://github.com/buggregator/frontend) releases
2. **PHP VarDumper parser**: Builds a self-contained PHP binary (`micro.sfx` + Composer deps + parser script) via [static-php-cli](https://github.com/crazywhalecc/static-php-cli)
3. **Go binary**: Compiles everything into a single binary with `go:embed`

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
  - event: "*"                          # "*" for all, or specific type
    url: https://slack.example.com/webhook
    headers:
      Authorization: "Bearer token"
    verify_ssl: false                   # default: false
    retry: true                         # retry up to 3x with backoff (default: true)
  - event: sentry
    url: https://pagerduty.example.com/alert

# Pre-defined projects
projects:
  - key: my-app
    name: My Application
  - key: staging
    name: Staging
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
| `METRICS_ADDR` | — | Separate metrics server address (e.g., `:9090`) |

Config file values support `${VAR}` and `${VAR:default}` syntax for environment variable substitution.

## Architecture

```
┌─────────────────────────────────────┐
│          Single Go Binary           │
│                                     │
│  HTTP :8000                         │
│  ├── REST API (events CRUD)         │
│  ├── Ingestion Pipeline             │
│  │   ├── Sentry  (X-Sentry-Auth)    │
│  │   ├── Ray     (User-Agent: Ray)  │
│  │   ├── Inspector (X-Inspector-*)  │
│  │   ├── Profiler  (X-Profiler-*)   │
│  │   ├── SMS     (/sms endpoint)    │
│  │   └── HttpDump (catch-all)       │
│  ├── WebSocket (/connection/ws)     │
│  │   └── Centrifugo v5 protocol     │
│  └── Frontend (embedded SPA)        │
│                                     │
│  TCP :9912 — VarDumper → PHP parser │
│  TCP :9913 — Monolog (ndjson)       │
│  TCP :1025 — SMTP (go-smtp)        │
│                                     │
│  SQLite (in-memory or file)         │
└─────────────────────────────────────┘
```

### Event Detection

Events are routed to modules by:
1. **URI userinfo**: `http://sentry@host:8000` → Sentry module
2. **Headers**: `X-Buggregator-Event: profiler` → Profiler module
3. **Basic Auth**: `Authorization: Basic base64(type:project)`
4. **Module-specific**: `X-Sentry-Auth`, `User-Agent: Ray`, etc.

### Adding a New Module

1. Create `modules/yourtype/module.go`:

```go
type Module struct { module.BaseModule }
func New() *Module             { return &Module{} }
func (m *Module) Name() string { return "YourType" }
func (m *Module) Type() string { return "your-type" }
func (m *Module) HTTPHandler() module.HTTPIngestionHandler { return &handler{} }
func (m *Module) PreviewMapper() event.PreviewMapper       { return &preview{} }
```

2. Add SQL migrations in `modules/yourtype/migrations/`
3. Register in `cmd/buggregator/main.go`:
```go
registry.Register(yourtype.New())
```

### Migration System

Each module has a `migrations/` directory with SQL files (embedded via `go:embed`), sorted by filename:

```
modules/profiler/migrations/
├── 2024_01_01_000010_create_profiles_table.sql
└── 2024_01_01_000011_create_profile_edges_table.sql
```

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

## API

### Events
```
GET    /api/events               List events (?type=sentry&project=default)
GET    /api/events/preview       List with previews
GET    /api/event/{uuid}         Get single event
DELETE /api/event/{uuid}         Delete event
DELETE /api/events               Clear events
POST   /api/event/{uuid}/pin     Pin event
DELETE /api/event/{uuid}/pin     Unpin event
```

### Profiler
```
GET /api/profiler/{uuid}/summary
GET /api/profiler/{uuid}/call-graph?metric=cpu&threshold=1&percentage=10
GET /api/profiler/{uuid}/top?limit=100&metric=cpu
GET /api/profiler/{uuid}/flame-chart?metric=wt
```

### Other
```
GET /api/version                 Server version
GET /api/settings                Settings and enabled modules
GET /api/projects                Project list
GET /connection/websocket        WebSocket (Centrifugo v5 protocol)
```

## Dependencies

| Dependency | Purpose |
|-----------|---------|
| [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) | Pure Go SQLite (no CGO) |
| [nhooyr.io/websocket](https://pkg.go.dev/nhooyr.io/websocket) | WebSocket server |
| [go-smtp](https://github.com/emersion/go-smtp) | SMTP server |
| [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) | YAML config |
| [prometheus/client_golang](https://github.com/prometheus/client_golang) | Prometheus metrics |

## License

MIT
