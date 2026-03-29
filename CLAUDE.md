# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build          # Full build: downloads frontend, builds PHP parser, compiles binary
make run            # Build and run
go build ./...      # Quick Go compilation check
go test ./...       # Run all tests
go vet ./...        # Lint (also used in CI)
```

Single test: `go test ./internal/server/http/ -run TestAPI_Version`

Cross-compile: `make build-cross GOOS=darwin GOARCH=arm64`

The frontend is a pre-built SPA downloaded during `make build` and embedded via `go:embed` in `internal/frontend/`.

## Architecture

Buggregator is a debugging server that captures events from various SDKs (Sentry, Ray, Monolog, etc.) over HTTP and TCP,
stores them in SQLite, and broadcasts to connected WebSocket clients in real time.

### Startup & Wiring

Entry point is `cmd/buggregator/main.go`. It creates all modules, registers them with a `module.Registry`, builds an
`EventService`, injects it into TCP modules via `SetEventService()`, then passes everything to `app.New()` which
orchestrates startup in `app.Run()`.

Two `EventService` instances exist: one in `main.go` (for TCP modules) and one in `app.go` (for the HTTP ingestion
pipeline). Both share the same store and hub.

### Module System

Every module implements `module.Module` (`internal/module/module.go`). Use `BaseModule` for no-op defaults. Modules are
either HTTP-based or TCP-based:

- **HTTP modules** return an `HTTPIngestionHandler` with `Priority()`, `Match(*http.Request)`, and
  `Handle(*http.Request)`. Lower priority runs first. The http-dump module has priority 9999 as the catch-all.
- **TCP modules** return `[]tcp.ServerConfig` with either a `ConnectionHandler` (generic accept loop) or a `Starter` (
  self-managed, like SMTP's go-smtp server).

Registration order matters: http-dump must be registered last.

### Ingestion Pipeline

`internal/server/http/ingestion.go` — the pipeline for HTTP events:

1. **Detect** (`detect.go`): Extract event type/project from URI userinfo (`sentry@host`), headers (
   `X-Buggregator-Event`), or Basic Auth
2. **Match**: Try handlers in priority order; first match wins
3. **Handle**: Parse request body → return `event.Incoming`
4. **Store & Broadcast** (`event_service.go`): Store in SQLite → build preview → broadcast via WebSocket hub → notify
   modules (`OnEventStored`)

Unmatched requests fall through to the embedded frontend SPA.

### WebSocket Protocol

The hub (`internal/server/ws/hub.go`) implements the Centrifugo v5 JSON protocol. Clients auto-subscribe to
`events.project.{key}` channels. RPC calls are bridged to HTTP handlers via `MuxRPCHandler`.

### Storage

`event.Store` interface in `internal/event/store.go`. Single implementation: `SQLiteStore` in
`internal/storage/sqlite.go` (pure Go SQLite via `modernc.org/sqlite`, single-writer mode).

Migrations live in each module and run via `storage.Migrator`. Attachments use a pluggable `AttachmentStore` (in-memory
or filesystem).

### Metrics

`internal/metrics/` provides optional Prometheus instrumentation. Enabled via config (`metrics.enabled: true`).
`InstrumentedStore` decorates `event.Store`. HTTP middleware captures request metrics. The hub and TCP manager report
connection gauges.

### Config Priority

CLI flags > environment variables > `buggregator.yaml` > defaults. YAML supports `${VAR:default}` substitution. See
`buggregator.yaml.example`.

## Test Patterns

Tests use `storage.Open(":memory:")` for in-memory SQLite, `t.Helper()` setup functions, `t.Cleanup()` for teardown, and
table-driven subtests. Handler tests call module methods directly with `httptest.NewRequest()` and check the returned
`*event.Incoming`.

Pass `nil` for optional dependencies like `*metrics.Collector` in tests.
