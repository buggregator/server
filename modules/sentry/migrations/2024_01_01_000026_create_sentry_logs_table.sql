CREATE TABLE IF NOT EXISTS sentry_logs (
    id               TEXT PRIMARY KEY,
    trace_id         TEXT,
    span_id          TEXT,
    level            TEXT NOT NULL,
    severity_number  INTEGER,
    body             TEXT NOT NULL,
    attributes       TEXT,
    log_ts           TEXT,
    received_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_sentry_logs_trace ON sentry_logs(trace_id) WHERE trace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sentry_logs_level ON sentry_logs(level);
CREATE INDEX IF NOT EXISTS idx_sentry_logs_ts    ON sentry_logs(log_ts);
