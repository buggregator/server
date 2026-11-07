-- Mock response templates (#4493): a rule's body and header values are Go
-- templates when template = 1; seed <> 0 fixes its fake data.
ALTER TABLE mock_rules ADD COLUMN template INTEGER NOT NULL DEFAULT 0;
ALTER TABLE mock_rules ADD COLUMN seed INTEGER NOT NULL DEFAULT 0;
