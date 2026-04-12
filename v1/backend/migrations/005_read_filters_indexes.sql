-- schema_version 005: indexes for fast filtered reads
CREATE INDEX IF NOT EXISTS idx_nodes_task_path ON nodes(task_id, path) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_nodes_task_updated ON nodes(task_id, updated_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_nodes_task_depth ON nodes(task_id, depth) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_nodes_task_kind_status ON nodes(task_id, kind, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_events_task_type_created ON events(task_id, type, created_at);
