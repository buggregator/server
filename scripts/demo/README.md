# Demo data

Fills a Buggregator server with realistic demo data of every event type, for
screenshots, UI work and trying the product out. The data belongs to two
made-up apps: an online shop (project `shop`) and its billing service
(project `billing`). All names, emails (`@example.com`), phone numbers
(`+1 555 01xx`) and IPs (documentation ranges) are fake.

## Run

Fresh server with demo data, stopped with Ctrl-C (needs Go; Docker for the
VarDumper part):

```bash
scripts/demo/run.sh                     # builds the server from this checkout
scripts/demo/run.sh ./bin/buggregator   # or use a built binary (or BUGGREGATOR_BIN)
```

Then open http://127.0.0.1:8000. Ports can be changed with `HTTP_PORT` (8000),
`SMTP_PORT` (1025), `MONOLOG_PORT` (9913), `VARDUMPER_PORT` (9912) and
`PROXY_PORT` (8080). `SKIP_VARDUMPER=1` skips the Docker step.

Seed a server that is already running:

```bash
go run ./scripts/demo/seed -http http://127.0.0.1:8000 -smtp 127.0.0.1:1025 -monolog 127.0.0.1:9913
```

Running the seed again adds more data. Mock rules that already exist are reused.
Timestamps are relative to now (within the last hour), so the Sentry service map
(60-minute window) shows everything right after seeding.

## What appears

| Type | Data |
|------|------|
| Sentry | Errors grouped with repeats (duplicate order, out of stock, declined card, tax timeout, ...), with stack traces, request, user, tags, breadcrumbs. About 45 transactions with db, cache, http.client and queue spans; the shop's checkout continues as a billing trace. Logs and trace metrics (counters, gauges, distributions). |
| Ray | Log with color, table, SQL query, measure, label, exception, notification. |
| Inspector | Two shop requests and a billing queue job with segments. |
| Profiler | Two shop catalog profiles and two billing charge profiles (Laravel call trees). |
| SMS | Twilio, MessageBird, Vonage and Clickatell messages. |
| HTTP dumps | GET/PUT/POST/PATCH/DELETE requests with JSON, form and query data, and a multipart upload (CSV + PDF). |
| Mock server | Six rules (one returns 503, one has a delay, one serves every project) and nine hits, one of them unmatched. |
| SMTP | Order confirmation and password reset (HTML + text), an invoice with CC and two attachments, a payment-failed mail with BCC. |
| Monolog | Nine records from DEBUG to CRITICAL with context and extra. |
| VarDumper | Objects, arrays, scalars and SQL/JSON code dumps (see below). |

## VarDumper

VarDumper payloads are PHP-serialized objects, so the Go seed cannot make them.
`php/dumps.php` calls Symfony's VarDumper server connection the way `dump()`
does in an app. `run.sh` runs it in Docker. By hand:

```bash
docker build -t buggregator-demo-dumps scripts/demo/php
docker run --rm --add-host=host.docker.internal:host-gateway \
  -e VAR_DUMPER_SERVER=host.docker.internal:9912 buggregator-demo-dumps
```

The server's VarDumper port must be reachable from the container: listen on
`0.0.0.0:9912`, not `127.0.0.1`. With PHP and Composer installed, run it
directly: `composer require symfony/var-dumper` in `php/`, then
`VAR_DUMPER_SERVER=127.0.0.1:9912 php dumps.php`.
