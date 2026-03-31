CREATE TABLE IF NOT EXISTS sentry_exceptions (
    id              TEXT PRIMARY KEY,
    error_event_id  TEXT NOT NULL,
    position        INTEGER NOT NULL DEFAULT 0,
    exception_type  TEXT,
    exception_value TEXT,
    mechanism_type  TEXT,
    handled         INTEGER,
    stacktrace      TEXT,
    FOREIGN KEY (error_event_id) REFERENCES sentry_error_events(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sentry_exc_event ON sentry_exceptions(error_event_id);
CREATE INDEX IF NOT EXISTS idx_sentry_exc_type  ON sentry_exceptions(exception_type);
