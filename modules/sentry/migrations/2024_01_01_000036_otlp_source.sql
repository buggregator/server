-- OpenTelemetry (OTLP) spans and logs land in the same tables as Sentry ones.
-- source tells them apart: "sentry" or "otel". Rows stored before are Sentry.
-- service_name on transactions and logs: the OTel resource service.name.
ALTER TABLE sentry_transactions ADD COLUMN source TEXT NOT NULL DEFAULT 'sentry';
ALTER TABLE sentry_transactions ADD COLUMN service_name TEXT;
ALTER TABLE sentry_spans ADD COLUMN source TEXT NOT NULL DEFAULT 'sentry';
ALTER TABLE sentry_logs ADD COLUMN source TEXT NOT NULL DEFAULT 'sentry';
ALTER TABLE sentry_logs ADD COLUMN service_name TEXT;
