-- Mock rule conditions (#512): a JSON list of {in, name, op, value} on
-- request headers and JSON body fields. An empty list: method, path and
-- project alone decide.
ALTER TABLE mock_rules ADD COLUMN conditions TEXT NOT NULL DEFAULT '[]';
