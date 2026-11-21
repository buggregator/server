-- User feedback sent by Sentry SDKs: the legacy "user_report" item and the
-- current "feedback" item (an event with contexts.feedback).
-- event_id: the error the feedback is about, empty when it stands alone
-- (e.g. a feedback widget). feedback_id: the id the SDK gave the feedback
-- itself (the feedback event's event_id), empty for user_report.
CREATE TABLE IF NOT EXISTS sentry_feedback (
    id           TEXT PRIMARY KEY,
    project_id   TEXT NOT NULL DEFAULT 'default',
    event_id     TEXT NOT NULL DEFAULT '',
    feedback_id  TEXT NOT NULL DEFAULT '',
    name         TEXT NOT NULL DEFAULT '',
    email        TEXT NOT NULL DEFAULT '',
    message      TEXT NOT NULL DEFAULT '',
    url          TEXT NOT NULL DEFAULT '',
    source       TEXT NOT NULL,
    payload      TEXT,
    received_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_sentry_feedback_event ON sentry_feedback(event_id) WHERE event_id <> '';
CREATE INDEX IF NOT EXISTS idx_sentry_feedback_project ON sentry_feedback(project_id, received_at);
