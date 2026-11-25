CREATE TABLE IF NOT EXISTS assist_threads (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    kind         TEXT NOT NULL,
    key          TEXT NOT NULL,
    title        TEXT NOT NULL DEFAULT '',
    project      TEXT NOT NULL DEFAULT '',
    conversation TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    UNIQUE (kind, key)
);
CREATE INDEX IF NOT EXISTS idx_assist_threads_conversation ON assist_threads (conversation);
CREATE INDEX IF NOT EXISTS idx_assist_threads_updated ON assist_threads (updated_at);

CREATE TABLE IF NOT EXISTS assist_messages (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    thread_id INTEGER NOT NULL,
    sender    TEXT NOT NULL,
    role      TEXT NOT NULL DEFAULT '',
    project   TEXT NOT NULL DEFAULT '',
    text      TEXT NOT NULL,
    context   INTEGER NOT NULL DEFAULT 0,
    at        TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_assist_messages_thread ON assist_messages (thread_id, id);
