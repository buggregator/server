-- OpenTelemetry (OTLP) spans and logs land in the same tables as Sentry ones.
-- source tells them apart: "sentry" or "otel". Rows stored before are Sentry.
-- service_name on transactions and logs: the OTel resource service.name.
ALTER TABLE sentry_transactions ADD COLUMN source TEXT NOT NULL DEFAULT 'sentry';
ALTER TABLE sentry_transactions ADD COLUMN service_name TEXT;
ALTER TABLE sentry_spans ADD COLUMN source TEXT NOT NULL DEFAULT 'sentry';
ALTER TABLE sentry_logs ADD COLUMN source TEXT NOT NULL DEFAULT 'sentry';
ALTER TABLE sentry_logs ADD COLUMN service_name TEXT;
-- parent_span_id of a transaction: empty for the trace root. The traces list
-- shows one row per trace, its root, picked from this column, not by reading
-- payloads on every load.
ALTER TABLE sentry_transactions ADD COLUMN parent_span_id TEXT;
UPDATE sentry_transactions SET parent_span_id = COALESCE(
    NULLIF(json_extract(payload, '$.contexts.trace.parent_span_id'), ''),
    NULLIF(json_extract(payload, '$.parent_span_id'), ''))
WHERE json_valid(payload);
CREATE INDEX IF NOT EXISTS idx_sentry_txn_trace_root ON sentry_transactions(trace_id, (parent_span_id IS NOT NULL), start_ts);
