CREATE TABLE IF NOT EXISTS sentry_breadcrumbs (
    id              TEXT PRIMARY KEY,
    error_event_id  TEXT NOT NULL,
    bc_type         TEXT,
    category        TEXT,
    level           TEXT,
    message         TEXT,
    bc_timestamp    TEXT,
    data            TEXT,
    FOREIGN KEY (error_event_id) REFERENCES sentry_error_events(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sentry_bc_event ON sentry_breadcrumbs(error_event_id);
