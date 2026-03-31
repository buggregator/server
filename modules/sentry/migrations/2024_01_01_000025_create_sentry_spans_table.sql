CREATE TABLE IF NOT EXISTS sentry_spans (
    id             TEXT PRIMARY KEY,
    span_id        TEXT NOT NULL,
    transaction_id TEXT,
    trace_id       TEXT NOT NULL,
    parent_span_id TEXT,
    op             TEXT,
    description    TEXT,
    status         TEXT,
    start_ts       TEXT,
    end_ts         TEXT,
    duration_ms    INTEGER,
    service_name   TEXT,
    peer_address   TEXT,
    peer_type      TEXT,
    is_error       INTEGER NOT NULL DEFAULT 0,
    attributes     TEXT,
    FOREIGN KEY (transaction_id) REFERENCES sentry_transactions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sentry_spans_transaction ON sentry_spans(transaction_id);
CREATE INDEX IF NOT EXISTS idx_sentry_spans_trace       ON sentry_spans(trace_id);
CREATE INDEX IF NOT EXISTS idx_sentry_spans_span_id     ON sentry_spans(span_id);
CREATE INDEX IF NOT EXISTS idx_sentry_spans_peer        ON sentry_spans(peer_address) WHERE peer_address IS NOT NULL;
