-- Who made a stored mock rule (#4493): "api" (UI, REST) or "mcp" (an agent).
ALTER TABLE mock_rules ADD COLUMN source TEXT NOT NULL DEFAULT 'api';
