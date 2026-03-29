CREATE TABLE IF NOT EXISTS events (
    uuid TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    payload TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    project TEXT,
    is_pinned INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS events_idx_type ON events(type);
CREATE INDEX IF NOT EXISTS events_idx_project ON events(project);
