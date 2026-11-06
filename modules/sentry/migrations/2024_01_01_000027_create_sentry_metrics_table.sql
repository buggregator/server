CREATE TABLE IF NOT EXISTS sentry_metrics (
    id           TEXT PRIMARY KEY,
    trace_id     TEXT,
    span_id      TEXT,
    name         TEXT NOT NULL,
    type         TEXT NOT NULL,
    value        REAL NOT NULL,
    unit         TEXT,
    attributes   TEXT,
    metric_ts    TEXT,
    received_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_sentry_metrics_name  ON sentry_metrics(name);
CREATE INDEX IF NOT EXISTS idx_sentry_metrics_trace ON sentry_metrics(trace_id) WHERE trace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sentry_metrics_ts    ON sentry_metrics(metric_ts);
