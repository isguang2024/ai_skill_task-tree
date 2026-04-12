-- schema_version 011: add execution_log to memory tables
ALTER TABLE node_memory_current ADD COLUMN execution_log TEXT NOT NULL DEFAULT '';
