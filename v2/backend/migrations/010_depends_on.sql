-- schema_version 010: node dependency relationships
ALTER TABLE nodes ADD COLUMN depends_on_json TEXT NOT NULL DEFAULT '[]';
