-- Folders added to the code graph from the UI (inside code_graph.base_dir).
-- Config roots are not stored here.
CREATE TABLE IF NOT EXISTS code_ui_roots (
    name        TEXT PRIMARY KEY,
    path        TEXT NOT NULL,
    added_at    TEXT NOT NULL DEFAULT ''
);
