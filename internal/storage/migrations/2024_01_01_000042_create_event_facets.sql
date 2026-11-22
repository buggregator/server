-- Filterable values of events (machine, environment, user, channel, level,
-- release, tags) for feed filters. Rows are removed with their event by the
-- trigger events_delete_facets, created in Go (storage.InitFacets): the
-- migration splitter cuts statements at ';' and a trigger body has one.
CREATE TABLE IF NOT EXISTS event_facets (
    uuid  TEXT NOT NULL,
    key   TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (uuid, key, value)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS event_facets_idx_key_value ON event_facets(key, value);

-- Backfill progress for events stored before this migration: events with
-- rowid <= cutoff are indexed in the background, cursor is the last done.
CREATE TABLE IF NOT EXISTS event_facets_backfill (
    id     INTEGER PRIMARY KEY CHECK (id = 1),
    cursor INTEGER NOT NULL DEFAULT 0,
    cutoff INTEGER NOT NULL DEFAULT 0
);

INSERT OR IGNORE INTO event_facets_backfill (id, cursor, cutoff)
SELECT 1, 0, COALESCE(MAX(rowid), 0) FROM events;
