---
name: sentry-sdk-testing
description: Test Buggregator's Sentry module compatibility with different Sentry PHP SDK versions. Triggers when the user asks to test Sentry compatibility, verify a new SDK version works, check Sentry transport changes, or mentions "sentry sdk", "sentry version", "sentry compatibility". Also triggers on requests to prepare for a new Sentry SDK release.
---

# Sentry SDK Compatibility Testing

## Role

You are a protocol compatibility engineer. Your job is to verify that Buggregator's Sentry ingestion pipeline correctly handles payloads from all supported Sentry PHP SDK versions (v2, v3, v4, and future releases).

## When to Use

- A new major/minor Sentry PHP SDK version is released
- A user reports that events from a specific SDK version are not appearing in the UI
- Before a Buggregator release, to verify no regressions in Sentry support

## Workflow

### Phase 1 — Install SDK Versions

Create isolated temporary directories for each SDK version and install them via Composer:

```bash
# For each version you need to test:
mkdir -p /tmp/sentry-vN
cd /tmp/sentry-vN
composer require sentry/sentry:"^N.0" --no-interaction

# If PHP version constraint fails:
composer require sentry/sentry:"^N.0" --no-interaction --ignore-platform-reqs

# Verify installed version:
cd /tmp/sentry-vN && composer show sentry/sentry | head -5
```

### Phase 2 — Analyze Transport Layer

For each SDK version, examine these key files:

| What to Find | Where to Look |
|---|---|
| Endpoint paths | `src/Dsn.php` — `getStoreApiEndpointUrl()`, `getEnvelopeApiEndpointUrl()` |
| Transport logic | `src/Transport/HttpTransport.php` — `send()` method, which endpoint is chosen |
| Headers | `src/HttpClient/Authentication/`, `src/Util/Http.php` |
| Serialization | `src/Serializer/PayloadSerializer.php` (v2/v3) or `src/Serializer/EnvelopItems/*.php` (v4+) |
| Compression | `src/HttpClient/Plugin/GzipEncoderPlugin.php` or `src/HttpClient/HttpClient.php` |
| Event structure | `src/Event.php` — `toArray()` method (v2/v3) or `EnvelopItems/EventItem.php` (v4+) |

Document findings in a comparison table covering:
1. **Endpoint** — `/store/` vs `/envelope/` vs both
2. **Content-Type** — `application/json` vs `application/x-sentry-envelope`
3. **Body format** — plain JSON vs envelope (header + items)
4. **`event_id` location** — in payload, in envelope header, or both
5. **Timestamp format** — ISO 8601 string vs numeric epoch float
6. **Message format** — plain string vs `{"message","params","formatted"}` object
7. **New/removed fields** in event payload
8. **New envelope item types** (e.g., `log`, `trace_metric`, `client_report`)
9. **Compression** — gzip, deflate, or none by default

### Phase 3 — Generate Real Payloads

Write a PHP script in each SDK directory to produce real serialized output:

```php
<?php
require_once 'vendor/autoload.php';

use Sentry\Options;
use Sentry\Serializer\PayloadSerializer;
use Sentry\Event;
use Sentry\Severity;
use Sentry\ExceptionDataBag;

$options = new Options([
    'dsn' => 'http://test@localhost:8000/1',
    'http_compression' => false,
    // For v3 with tracing: 'traces_sample_rate' => 1.0,
]);

$serializer = new PayloadSerializer($options);

$event = Event::createEvent();       // v3/v4
// $event = new Event();             // v2
$event->setLevel(Severity::error());
$event->setEnvironment('production');
$event->setRelease('1.0.0');
$event->setServerName('web-01');

$exception = new ExceptionDataBag(new \RuntimeException('Test error'));
$event->setExceptions([$exception]);

echo $serializer->serialize($event);
```

Save generated payloads for use in Go tests.

### Phase 4 — Write Go Tests

Add tests in `modules/sentry/handler_test.go` using real payloads from Phase 3.

Each test must verify:

1. **`Handle()` returns non-nil** — the event is recognized
2. **`inc.UUID`** — matches the expected event_id
3. **`inc.Type == "sentry"`** — correct event type
4. **`inc.Project`** — extracted from URL path
5. **`event_id` in payload** — present in `inc.Payload` JSON (critical for frontend rendering)
6. **Exception/message data preserved** — key fields survive parsing

For structured storage tests (with DB), also verify:
- Row exists in `sentry_error_events`
- Exceptions stored in `sentry_exceptions`
- Breadcrumbs stored in `sentry_breadcrumbs`

Test naming convention:
```
TestHandler_SDKvN_Format       (e.g., TestHandler_SDKv4_Envelope)
TestHandler_SDKvN_Format_WithDB (e.g., TestHandler_SDKv2_PlainJSON_WithDB)
```

### Phase 5 — Fix Compatibility Issues

Common patterns that break between SDK versions:

| Problem | Symptom | Fix Pattern |
|---|---|---|
| `event_id` missing from item payload | Events invisible in UI | Inject from envelope header via `injectEventID()` |
| Timestamp format change (string vs float) | `json.Unmarshal` fails silently | Use `FlexibleTS` type that accepts both |
| Field type change (string vs object) | Structured storage skipped | Use flexible unmarshaler (e.g., `FlexibleMessage`) |
| New envelope item type | Unknown items silently dropped | Add case to `handleEnvelope()` switch |
| New fields in payload | Data loss in structured tables | Add fields to Go structs, update store functions |
| Endpoint change | Handler doesn't match request | Update `Match()` path checks |

### Phase 6 — Verify All Tests Pass

```bash
# Sentry module tests only:
cd /home/butschster/repos/buggregator/server
go test ./modules/sentry/ -v

# Full project:
go test ./...
```

## Known Version Differences (Reference)

### SDK v2 (2.x)
- Plain JSON only, `/api/{id}/store/`
- Timestamp: ISO 8601 string
- No envelope support
- No tracing/spans

### SDK v3 (3.x)
- Plain JSON to `/store/` (no tracing) or Envelope to `/envelope/` (with tracing)
- Timestamp: float epoch
- `event_id` present in both envelope header and item payload

### SDK v4 (4.x)
- Always Envelope to `/api/{id}/envelope/`
- Timestamp: float epoch
- `event_id` ONLY in envelope header (not in item payload)
- New item types: `log`, `trace_metric`, `client_report`
- SDK payload includes `packages` array
- Span `origin` field added

## Key Files in Buggregator

| File | Responsibility |
|---|---|
| `modules/sentry/handler.go` | Request matching, decompression, JSON vs envelope routing, event_id injection |
| `modules/sentry/envelope.go` | Envelope parsing (header + item pairs) |
| `modules/sentry/types.go` | `ErrorEvent`, `FlexibleTS`, `FlexibleMessage`, `Transaction`, `RawSpan`, etc. |
| `modules/sentry/store_error.go` | Structured storage for error events, exceptions, breadcrumbs |
| `modules/sentry/store_transaction.go` | Structured storage for transactions and spans |
| `modules/sentry/preview.go` | Preview mapper for WebSocket broadcast |
| `modules/sentry/handler_test.go` | SDK compatibility tests |

## Cleanup

After testing, remove temporary directories:

```bash
rm -rf /tmp/sentry-v2 /tmp/sentry-v3 /tmp/sentry-v4 /tmp/sentry-v5
```
