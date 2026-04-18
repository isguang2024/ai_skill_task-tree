ALTER TABLE node_runs ADD COLUMN usage_thread_id TEXT;
ALTER TABLE node_runs ADD COLUMN usage_start_tokens INTEGER NOT NULL DEFAULT 0;
