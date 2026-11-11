-- Errors from a standard DSN (/api/1/...) were stored with project_id NULL,
-- while the main events table stores them under "default". Use the same key,
-- so ?project=default finds them.
UPDATE sentry_error_events SET project_id = 'default' WHERE project_id IS NULL OR project_id = '';
