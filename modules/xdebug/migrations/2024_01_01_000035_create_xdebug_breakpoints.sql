CREATE TABLE IF NOT EXISTS xdebug_breakpoints (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    type        TEXT NOT NULL DEFAULT 'line',
    file        TEXT NOT NULL DEFAULT '',
    line        INTEGER NOT NULL DEFAULT 0,
    exception   TEXT NOT NULL DEFAULT '',
    condition   TEXT NOT NULL DEFAULT '',
    enabled     INTEGER NOT NULL DEFAULT 1,
    source      TEXT NOT NULL DEFAULT 'ui',
    created_at  TEXT NOT NULL
);
