-- Full-text search over the text a module indexes for an event (Monolog:
-- message, channel, context, extra). Trigram tokens find any substring of
-- three or more characters, case-insensitive, like grep.
--
-- event_search_ids gives each indexed event a stable integer id (events has
-- no INTEGER PRIMARY KEY, so its rowid may change on VACUUM). That id is the
-- rowid of the event's row in event_search. Both rows go with their event
-- by the trigger events_delete_search, created in Go (storage.InitSearch):
-- the migration splitter cuts statements at semicolons, even in comments.
CREATE TABLE IF NOT EXISTS event_search_ids (
    id   INTEGER PRIMARY KEY,
    uuid TEXT NOT NULL UNIQUE
);

CREATE VIRTUAL TABLE IF NOT EXISTS event_search USING fts5(body, tokenize = 'trigram');

-- Backfill progress for events stored before this migration (see
-- event_facets_backfill).
CREATE TABLE IF NOT EXISTS event_search_backfill (
    id     INTEGER PRIMARY KEY CHECK (id = 1),
    cursor INTEGER NOT NULL DEFAULT 0,
    cutoff INTEGER NOT NULL DEFAULT 0
);

INSERT OR IGNORE INTO event_search_backfill (id, cursor, cutoff)
SELECT 1, 0, COALESCE(MAX(rowid), 0) FROM events;
