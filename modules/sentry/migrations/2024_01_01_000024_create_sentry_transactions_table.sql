CREATE TABLE IF NOT EXISTS sentry_transactions (
    id               TEXT PRIMARY KEY,
    event_id         TEXT NOT NULL UNIQUE,
    trace_id         TEXT NOT NULL,
    transaction_name TEXT NOT NULL,
    op               TEXT,
    status           TEXT,
    start_ts         TEXT,
    end_ts           TEXT,
    duration_ms      INTEGER,
    environment      TEXT,
    release          TEXT,
    measurements     TEXT,
    payload          TEXT NOT NULL,
    FOREIGN KEY (trace_id) REFERENCES sentry_traces(trace_id)
);

CREATE INDEX IF NOT EXISTS idx_sentry_txn_trace ON sentry_transactions(trace_id);
CREATE INDEX IF NOT EXISTS idx_sentry_txn_start ON sentry_transactions(start_ts);
CREATE INDEX IF NOT EXISTS idx_sentry_txn_name  ON sentry_transactions(transaction_name);
