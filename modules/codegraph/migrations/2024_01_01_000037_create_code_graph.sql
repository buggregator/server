CREATE TABLE IF NOT EXISTS code_roots (
    name        TEXT PRIMARY KEY,
    path        TEXT NOT NULL DEFAULT '',
    indexed_at  TEXT NOT NULL DEFAULT '',
    error       TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS code_files (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    root        TEXT NOT NULL,
    path        TEXT NOT NULL,
    lang        TEXT NOT NULL DEFAULT '',
    size        INTEGER NOT NULL DEFAULT 0,
    mtime       INTEGER NOT NULL DEFAULT 0,
    hash        TEXT NOT NULL DEFAULT '',
    parse_error INTEGER NOT NULL DEFAULT 0,
    UNIQUE (root, path)
);

CREATE TABLE IF NOT EXISTS code_symbols (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    root        TEXT NOT NULL,
    file_id     INTEGER NOT NULL,
    parent_id   INTEGER,
    kind        TEXT NOT NULL,
    name        TEXT NOT NULL,
    fqn         TEXT NOT NULL,
    line_start  INTEGER NOT NULL DEFAULT 0,
    line_end    INTEGER NOT NULL DEFAULT 0,
    signature   TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_code_symbols_fqn ON code_symbols (root, fqn);
CREATE INDEX IF NOT EXISTS idx_code_symbols_name ON code_symbols (name);
CREATE INDEX IF NOT EXISTS idx_code_symbols_file ON code_symbols (file_id);
CREATE INDEX IF NOT EXISTS idx_code_symbols_parent ON code_symbols (parent_id);

CREATE TABLE IF NOT EXISTS code_edges (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    root        TEXT NOT NULL,
    file_id     INTEGER NOT NULL,
    from_id     INTEGER,
    kind        TEXT NOT NULL,
    to_name     TEXT NOT NULL DEFAULT '',
    to_fqn      TEXT NOT NULL DEFAULT '',
    alt_fqn     TEXT NOT NULL DEFAULT '',
    to_kind     TEXT NOT NULL DEFAULT '',
    to_id       INTEGER,
    line        INTEGER NOT NULL DEFAULT 0,
    confidence  TEXT NOT NULL DEFAULT 'exact'
);

CREATE INDEX IF NOT EXISTS idx_code_edges_from ON code_edges (from_id);
CREATE INDEX IF NOT EXISTS idx_code_edges_to ON code_edges (to_id);
CREATE INDEX IF NOT EXISTS idx_code_edges_file ON code_edges (file_id);
CREATE INDEX IF NOT EXISTS idx_code_edges_name ON code_edges (root, to_name, confidence);
CREATE INDEX IF NOT EXISTS idx_code_edges_fqn ON code_edges (root, to_fqn);
