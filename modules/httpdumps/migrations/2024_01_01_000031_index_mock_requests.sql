-- /api/mocks/requests lists captured mock requests newest first. Without an
-- index it read every http-dump payload to test json_type. This partial index
-- holds only the mock requests, ordered by time. The WHERE must stay the same
-- as in listMockRequests for SQLite to use it.
CREATE INDEX IF NOT EXISTS idx_events_mock_requests ON events(timestamp)
    WHERE type = 'http-dump' AND json_type(payload, '$.mock') IS NOT NULL;
