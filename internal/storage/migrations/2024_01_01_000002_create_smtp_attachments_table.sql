CREATE TABLE IF NOT EXISTS smtp_attachments (
    uuid TEXT PRIMARY KEY,
    event_uuid TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    mime TEXT NOT NULL DEFAULT 'application/octet-stream',
    content_id TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS smtp_attachments_idx_event ON smtp_attachments(event_uuid);
