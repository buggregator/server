-- Traces data had no project, so clearing one project's Sentry data could only
-- wipe every project's. Rows stored before carry no project: put them under
-- "default", the key errors from a standard DSN use.
ALTER TABLE sentry_transactions ADD COLUMN project_id TEXT;
ALTER TABLE sentry_spans ADD COLUMN project_id TEXT;
ALTER TABLE sentry_logs ADD COLUMN project_id TEXT;
ALTER TABLE sentry_metrics ADD COLUMN project_id TEXT;

UPDATE sentry_transactions SET project_id = 'default' WHERE project_id IS NULL;
UPDATE sentry_spans SET project_id = 'default' WHERE project_id IS NULL;
UPDATE sentry_logs SET project_id = 'default' WHERE project_id IS NULL;
UPDATE sentry_metrics SET project_id = 'default' WHERE project_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_sentry_errors_project ON sentry_error_events(project_id);
CREATE INDEX IF NOT EXISTS idx_sentry_txn_project ON sentry_transactions(project_id);
CREATE INDEX IF NOT EXISTS idx_sentry_spans_project ON sentry_spans(project_id);
CREATE INDEX IF NOT EXISTS idx_sentry_logs_project ON sentry_logs(project_id);
CREATE INDEX IF NOT EXISTS idx_sentry_metrics_project ON sentry_metrics(project_id);
