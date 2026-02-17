-- If officers table was created from an older schema (e.g. database_schema.sql) without authority_level,
-- run this after 0002_authority_pilot_tables.sql. Skip if authority_level already exists.

ALTER TABLE officers ADD COLUMN authority_level TINYINT NOT NULL DEFAULT 1 AFTER location_id;
