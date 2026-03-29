CREATE TABLE IF NOT EXISTS projects (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT OR IGNORE INTO projects (key, name) VALUES ('default', 'Default');
