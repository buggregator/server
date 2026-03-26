# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Buggregator is a lightweight, standalone debugging server for PHP applications. It aggregates multiple debugging tools (Sentry, Ray, VarDumper, Monolog, SMTP, HTTP dumps, XHProf profiling, Inspector) into a single self-hosted platform with a real-time web UI.

Built on **Spiral Framework** with **RoadRunner** (async PHP app server), **Cycle ORM**, and **Centrifugo** (WebSocket).

## Commands

### Testing
```bash
vendor/bin/phpunit                              # Run all tests
vendor/bin/phpunit --testsuite="Feature tests"  # Feature tests only
vendor/bin/phpunit --testsuite="Unit tests"     # Unit tests only
vendor/bin/phpunit --filter=SomeTestClass        # Run a single test class
vendor/bin/phpunit --filter=testMethodName       # Run a single test method
```
Tests use SQLite in-memory database (configured in phpunit.xml).

### Code Quality
```bash
composer cs-check    # Check code style (PHP-CS-Fixer, PER-CS2.0)
composer cs-fix      # Fix code style
composer psalm       # Static analysis (Psalm, level 4)
composer rector      # Check automated refactoring (dry-run)
composer refactor    # Apply automated refactoring (Rector)
composer deptrack    # Check architectural dependency rules (Deptrac)
```

### Docker Development
```bash
make up       # Build and start all containers
make start    # Start containers
make stop     # Stop containers
make down     # Tear down containers
make bash     # Shell into server container
make recreate-db  # Recreate database
```

### Binary Downloads
```bash
composer download   # Download RoadRunner, Centrifugo, Dolt binaries via dload
```

## Architecture

### Namespaces & Autoloading
- `App\` → `app/src/` — Core application code
- `Modules\` → `app/modules/` — Feature modules
- `Database\` → `app/database/` — Migrations and factories
- `Tests\` → `tests/`

### Core Application (`app/src/`)
- **`Application/Kernel.php`** — Entry point; defines bootloader loading order (service providers)
- **`Application/Bootloader/`** — Service registration (DI bindings, routes, middleware, auth, broadcasting)
- **`Application/HTTP/`** — Controllers, middleware, interceptors
- **`Application/Service/`** — Business logic services
- **`Application/Persistence/`** — Data access layer
- **`Application/TCP/`** — TCP server handlers (VarDumper, Monolog, SMTP)
- **`Application/Broadcasting/`** — WebSocket channel definitions
- **`Interfaces/Http/`** — HTTP action handlers (thin controllers)
- **`Interfaces/Console/`** — CLI commands
- **`Interfaces/Centrifugo/`** — WebSocket RPC handlers
- **`Integration/`** — External service implementations (Auth0, Kinde, CycleOrm)

### Feature Modules (`app/modules/`)
Each module is self-contained with a consistent structure:
- `Application/` — Bootloader, command/query handlers, business logic
- `Domain/` — Entities, repository interfaces, value objects
- `Interfaces/` — HTTP/TCP handlers specific to the module
- `Integration/` — Repository implementations, external adapters
- `Exceptions/` — Module-specific exceptions

**Modules:** Events (core storage), Sentry, Ray, VarDumper, Monolog, Smtp, HttpDumps, Inspector, Profiler, Projects, Webhooks, Metrics

### Key Patterns
- **CQRS** — Commands for mutations, Queries for reads, dispatched via `CommandBusInterface`/`QueryBusInterface`
- **Bootloaders** — Spiral's service provider pattern; each module registers its own bootloader in `Kernel.php`
- **Interceptors** — Request/response processing pipeline (JSON resources, UUID conversion, type casting)
- **Domain Events** — Event-driven communication between modules
- **Deptrac** enforces module isolation and layered architecture rules

### Testing
- Base class: `Tests\TestCase` (extends Spiral's testing framework)
- `Tests\DatabaseTestCase` for tests needing database
- Helpers: `BroadcastFaker`, `EventsMocker`, `SpyConsoleInvoker`
- CQRS helpers: `$this->dispatchCommand()`, `$this->dispatchQuery()`
- Container access: `$this->get(SomeInterface::class)`

### Infrastructure
- **RoadRunner** serves HTTP, TCP (Monolog:9913, VarDumper:9912, SMTP:1025), and job queues
- **Centrifugo** handles WebSocket connections for real-time UI updates
- **Docker** setup uses Traefik reverse proxy, PostgreSQL, and a demo Laravel app
- Config: `.rr.yaml` (local), `.rr-prod.yaml` (production)
- Database: PostgreSQL/MySQL in production, SQLite for tests
