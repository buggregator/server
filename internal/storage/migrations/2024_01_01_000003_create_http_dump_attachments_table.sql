CREATE TABLE IF NOT EXISTS http_dump_attachments (
    uuid TEXT PRIMARY KEY,
    event_uuid TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    mime TEXT NOT NULL DEFAULT 'application/octet-stream'
);

CREATE INDEX IF NOT EXISTS http_dump_attachments_idx_event ON http_dump_attachments(event_uuid);
