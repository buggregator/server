-- Rules made by one OpenAPI import share a group (#4493), and a re-import
-- replaces the rules of the group.
ALTER TABLE mock_rules ADD COLUMN group_name TEXT NOT NULL DEFAULT '';
