-- SSO users, their sign-in sessions, per-user settings and the audit log.
-- Used only with auth on. SQLite foreign keys are off here: the code deletes
-- child rows itself.
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    provider      TEXT NOT NULL,
    subject       TEXT NOT NULL,
    email         TEXT NOT NULL DEFAULT '',
    name          TEXT NOT NULL DEFAULT '',
    avatar        TEXT NOT NULL DEFAULT '',
    hosted_domain TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL,
    last_login_at TEXT NOT NULL,
    UNIQUE (provider, subject)
);

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    revoked_at TEXT,
    user_agent TEXT NOT NULL DEFAULT '',
    ip         TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS sessions_idx_user ON sessions(user_id);

CREATE TABLE IF NOT EXISTS user_settings (
    user_id    TEXT NOT NULL,
    key        TEXT NOT NULL,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (user_id, key)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS audit_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    at         TEXT NOT NULL,
    user_id    TEXT NOT NULL DEFAULT '',
    user_label TEXT NOT NULL,
    action     TEXT NOT NULL,
    target     TEXT NOT NULL DEFAULT '',
    status     INTEGER NOT NULL DEFAULT 0,
    ip         TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS audit_log_idx_at ON audit_log(at);
