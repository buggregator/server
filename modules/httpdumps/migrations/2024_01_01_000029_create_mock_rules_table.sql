CREATE TABLE IF NOT EXISTS mock_rules (
    id          TEXT PRIMARY KEY,
    project     TEXT,
    name        TEXT NOT NULL DEFAULT '',
    method      TEXT NOT NULL,
    path        TEXT NOT NULL,
    status      INTEGER NOT NULL,
    headers     TEXT NOT NULL DEFAULT '{}',
    body        TEXT NOT NULL DEFAULT '',
    delay_ms    INTEGER NOT NULL DEFAULT 0,
    enabled     INTEGER NOT NULL DEFAULT 1,
    position    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mock_rules_position ON mock_rules(position);
