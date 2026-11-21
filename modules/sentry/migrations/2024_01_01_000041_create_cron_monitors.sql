-- Cron monitors: scheduled jobs that report check-ins, through the Sentry SDK
-- (check_in envelope item, cron HTTP endpoint) or the plain /ping URL.
-- Times are UTC text 2006-01-02T15:04:05.000Z. checkin_margin and
-- max_runtime are minutes. A monitor without schedule_type has no slots,
-- so it is never missed.
CREATE TABLE IF NOT EXISTS cron_monitors (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id       TEXT NOT NULL DEFAULT 'default',
    slug             TEXT NOT NULL,
    source           TEXT NOT NULL,
    schedule_type    TEXT NOT NULL DEFAULT '',
    schedule_value   TEXT NOT NULL DEFAULT '',
    schedule_unit    TEXT NOT NULL DEFAULT '',
    timezone         TEXT NOT NULL DEFAULT 'UTC',
    checkin_margin   INTEGER NOT NULL DEFAULT 1,
    max_runtime      INTEGER NOT NULL DEFAULT 30,
    last_status      TEXT NOT NULL DEFAULT '',
    last_checkin_at  TEXT,
    next_expected_at TEXT,
    created_at       TEXT NOT NULL,
    UNIQUE (project_id, slug)
);

-- One run of a monitor. server_made marks rows the server wrote itself
-- (missed, unknown), expected_at is then the slot that had no check-in.
CREATE TABLE IF NOT EXISTS cron_checkins (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id   INTEGER NOT NULL,
    check_in_id  TEXT NOT NULL,
    status       TEXT NOT NULL,
    started_at   TEXT NOT NULL,
    finished_at  TEXT,
    duration_ms  INTEGER,
    environment  TEXT NOT NULL DEFAULT '',
    trace_id     TEXT NOT NULL DEFAULT '',
    source       TEXT NOT NULL,
    body         TEXT NOT NULL DEFAULT '',
    server_made  INTEGER NOT NULL DEFAULT 0,
    expected_at  TEXT
);

CREATE INDEX IF NOT EXISTS idx_cron_checkins_monitor ON cron_checkins(monitor_id, started_at);
CREATE INDEX IF NOT EXISTS idx_cron_checkins_id ON cron_checkins(check_in_id);
CREATE INDEX IF NOT EXISTS idx_cron_checkins_open ON cron_checkins(status) WHERE status = 'in_progress';
CREATE INDEX IF NOT EXISTS idx_cron_checkins_started ON cron_checkins(started_at);

-- The last time the cron checker ran: a later start sees how long the
-- server was down and marks slots of that gap unknown, not missed.
CREATE TABLE IF NOT EXISTS cron_heartbeat (
    id       INTEGER PRIMARY KEY CHECK (id = 1),
    alive_at TEXT NOT NULL
);
