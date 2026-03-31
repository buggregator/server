CREATE TABLE IF NOT EXISTS sentry_error_events (
    id           TEXT PRIMARY KEY,
    event_id     TEXT NOT NULL UNIQUE,
    project_id   TEXT,
    trace_id     TEXT,
    span_id      TEXT,
    level        TEXT NOT NULL DEFAULT 'error',
    environment  TEXT,
    release      TEXT,
    server_name  TEXT,
    platform     TEXT,
    "transaction" TEXT,
    fingerprint  TEXT NOT NULL,
    handled      INTEGER,
    received_at  TEXT NOT NULL DEFAULT (datetime('now')),
    event_ts     TEXT,
    payload      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sentry_errors_fingerprint  ON sentry_error_events(fingerprint);
CREATE INDEX IF NOT EXISTS idx_sentry_errors_trace_id     ON sentry_error_events(trace_id) WHERE trace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sentry_errors_level        ON sentry_error_events(level);
CREATE INDEX IF NOT EXISTS idx_sentry_errors_received_at  ON sentry_error_events(received_at);
