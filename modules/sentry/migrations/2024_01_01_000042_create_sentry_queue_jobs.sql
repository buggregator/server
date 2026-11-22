-- One attempt of a queue job: a transaction with op queue.process (Sentry
-- QueueIntegration, or an OTel messaging span). message_id is the job's id,
-- shared by its attempts. attempt is messaging.message.retry.count + 1.
-- latency_ms is how long the job waited in the queue. Times are the
-- transaction's start_ts / end_ts (ISO, UTC).
CREATE TABLE IF NOT EXISTS sentry_queue_jobs (
    transaction_id TEXT PRIMARY KEY,
    message_id     TEXT,
    trace_id       TEXT NOT NULL,
    project_id     TEXT NOT NULL DEFAULT 'default',
    job            TEXT NOT NULL DEFAULT '',
    queue          TEXT NOT NULL DEFAULT '',
    connection     TEXT NOT NULL DEFAULT '',
    attempt        INTEGER NOT NULL DEFAULT 1,
    status         TEXT NOT NULL DEFAULT '',
    duration_ms    INTEGER,
    latency_ms     INTEGER,
    started_at     TEXT,
    finished_at    TEXT,
    FOREIGN KEY (transaction_id) REFERENCES sentry_transactions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sentry_queue_jobs_finished ON sentry_queue_jobs(project_id, finished_at);
CREATE INDEX IF NOT EXISTS idx_sentry_queue_jobs_message  ON sentry_queue_jobs(message_id);

-- Jobs stored before this table: read the same fields from the payload
-- (Sentry: contexts.trace.data, a segment: data or attributes).
-- sentry-laravel sends the latency in seconds, other SDKs in milliseconds.
INSERT OR IGNORE INTO sentry_queue_jobs
    (transaction_id, message_id, trace_id, project_id, job, queue, connection, attempt, status, duration_ms, latency_ms, started_at, finished_at)
SELECT id, NULLIF(msg, ''), trace_id, COALESCE(project_id, 'default'), COALESCE(transaction_name, ''),
       COALESCE(dest, ''), COALESCE(conn, ''), COALESCE(retry, 0) + 1, COALESCE(status, ''), duration_ms,
       CASE WHEN latency IS NULL THEN NULL WHEN sys = 'laravel' THEN CAST(ROUND(latency * 1000) AS INTEGER) ELSE CAST(ROUND(latency) AS INTEGER) END,
       start_ts, end_ts
FROM (
    SELECT id, trace_id, project_id, transaction_name, status, duration_ms, start_ts, end_ts,
        CAST(COALESCE(json_extract(payload, '$.contexts.trace.data."messaging.message.id"'), json_extract(payload, '$.data."messaging.message.id"'), json_extract(payload, '$.attributes."messaging.message.id"')) AS TEXT) AS msg,
        COALESCE(json_extract(payload, '$.contexts.trace.data."messaging.destination.name"'), json_extract(payload, '$.data."messaging.destination.name"'), json_extract(payload, '$.attributes."messaging.destination.name"')) AS dest,
        COALESCE(json_extract(payload, '$.contexts.trace.data."messaging.destination.connection"'), json_extract(payload, '$.data."messaging.destination.connection"'), json_extract(payload, '$.attributes."messaging.destination.connection"')) AS conn,
        CAST(COALESCE(json_extract(payload, '$.contexts.trace.data."messaging.message.retry.count"'), json_extract(payload, '$.data."messaging.message.retry.count"'), json_extract(payload, '$.attributes."messaging.message.retry.count"')) AS INTEGER) AS retry,
        CAST(COALESCE(json_extract(payload, '$.contexts.trace.data."messaging.message.receive.latency"'), json_extract(payload, '$.data."messaging.message.receive.latency"'), json_extract(payload, '$.attributes."messaging.message.receive.latency"')) AS REAL) AS latency,
        COALESCE(json_extract(payload, '$.contexts.trace.data."messaging.system"'), json_extract(payload, '$.data."messaging.system"'), json_extract(payload, '$.attributes."messaging.system"')) AS sys
    FROM sentry_transactions
    WHERE op = 'queue.process' AND json_valid(payload)
);
