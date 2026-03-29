# Go-Buggregator

A lightweight, standalone debugging server for PHP applications — packed into a single Go binary.

Drop-in replacement for [Buggregator](https://github.com/buggregator/server) that requires no PHP, no RoadRunner, no
Centrifugo, and no external database.

## Features

- **Single binary** — download and run, nothing else needed
- **SQLite in-memory** by default — no database setup, no leftover files
- **Embedded frontend** — web UI served from the binary itself
- **Real-time updates** — built-in WebSocket server
- **Modular architecture** — easy to add new event types

### Supported Protocols

| Protocol          | Transport | Port |
|-------------------|-----------|------|
| Sentry            | HTTP      | 8000 |
| Ray               | HTTP      | 8000 |
| Inspector         | HTTP      | 8000 |
| HTTP Dumps        | HTTP      | 8000 |
| Monolog           | TCP       | 9913 |
| SMTP              | TCP       | 1025 |
| Profiler (XHProf) | HTTP      | 8000 |

## Quick Start

```bash
# Build
make build

# Run (SQLite in-memory, all defaults)
./buggregator
```

The server starts on `http://localhost:8000` with all protocols ready.

## Build

```bash
# Download frontend + build binary
make build

# Download specific frontend version
make frontend FRONTEND_VERSION=1.28.0

# Just build (if frontend already downloaded)
go build -o buggregator ./cmd/buggregator
```

## Configuration

All options can be set via flags or environment variables:

| Flag            | Env            | Default    | Description                          |
|-----------------|----------------|------------|--------------------------------------|
| `-http-addr`    | `HTTP_ADDR`    | `:8000`    | HTTP listen address                  |
| `-db`           | `DATABASE_DSN` | `:memory:` | SQLite DSN (`:memory:` or file path) |
| `-smtp-addr`    | `SMTP_ADDR`    | `:1025`    | SMTP listen address                  |
| `-monolog-addr` | `MONOLOG_ADDR` | `:9913`    | Monolog TCP listen address           |

### Examples

```bash
# In-memory (default) — events lost on restart
./buggregator

# Persist to file
./buggregator -db ./debug.db

# Custom ports
./buggregator -http-addr :9000 -smtp-addr :2525
```

## REST API

```
GET    /api/version              # Server version
GET    /api/settings             # Server settings
GET    /api/events               # List events (?type=sentry&project=default)
GET    /api/events/preview       # List with previews
GET    /api/event/{uuid}         # Get single event
DELETE /api/event/{uuid}         # Delete event
DELETE /api/events               # Clear events
POST   /api/event/{uuid}/pin     # Pin event
DELETE /api/event/{uuid}/pin     # Unpin event
GET    /ws                       # WebSocket connection
```

### Profiler API

```
GET /api/profiler/{uuid}/summary
GET /api/profiler/{uuid}/call-graph
GET /api/profiler/{uuid}/top?limit=100
GET /api/profiler/{uuid}/flame-chart
```

## Architecture

```
┌─────────────────────────────────────┐
│          Single Go Binary           │
│                                     │
│  HTTP :8000                         │
│  ├── REST API (events CRUD)         │
│  ├── Ingestion pipeline             │
│  │   ├── Sentry  (X-Sentry-Auth)    │
│  │   ├── Ray     (User-Agent: Ray)  │
│  │   ├── Inspector (X-Inspector-*)  │
│  │   ├── Profiler  (X-Profiler-*)   │
│  │   └── HttpDumps (catch-all)      │
│  ├── WebSocket (/ws)                │
│  └── Frontend (embedded)            │
│                                     │
│  TCP :9913 — Monolog (ndjson)       │
│  TCP :1025 — SMTP (go-smtp)        │
│                                     │
│  SQLite (in-memory or file)         │
└─────────────────────────────────────┘
```

### Adding a New Module

1. Create `modules/yourtype/module.go`:

```go
package yourtype

import "github.com/buggregator/go-buggregator/internal/module"

type Module struct {
    module.BaseModule // no-op defaults for optional methods
}

func New() *Module             { return &Module{} }
func (m *Module) Name() string { return "YourType" }
func (m *Module) Type() string { return "your-type" }

func (m *Module) HTTPHandler() module.HTTPIngestionHandler {
    return &handler{}
}

func (m *Module) PreviewMapper() event.PreviewMapper {
    return &preview{}
}
```

2. Add migrations in `modules/yourtype/migrations/`:

```sql
-- 2024_06_01_000000_create_your_table.sql
CREATE TABLE IF NOT EXISTS your_table
(
    .
    .
    .
);
```

3. Register in `cmd/buggregator/main.go`:

```go
registry.Register(yourtype.New())
```

### Migration System

Each module can have a `migrations/` directory with SQL files embedded via `go:embed`. Files are named with date
prefixes for ordering:

```
modules/profiler/migrations/
├── 2024_01_01_000010_create_profiles_table.sql
└── 2024_01_01_000011_create_profile_edges_table.sql
```

All migrations from all modules are collected, sorted by filename, and executed at startup. A `migrations` table tracks
which have been applied.

## Dependencies

- [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) — pure Go SQLite (no CGO)
- [`nhooyr.io/websocket`](https://pkg.go.dev/nhooyr.io/websocket) — WebSocket
- [`github.com/emersion/go-smtp`](https://github.com/emersion/go-smtp) — SMTP server
- Go 1.22+ stdlib `net/http` — HTTP routing

## License

MIT
