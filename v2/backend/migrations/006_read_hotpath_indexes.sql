-- schema_version 006: extra indexes for cursor/order hot paths
CREATE INDEX IF NOT EXISTS idx_nodes_task_path_id ON nodes(task_id, path, id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_nodes_task_updated_id ON nodes(task_id, updated_at, id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_events_task_created_id ON events(task_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_events_node_created_id ON events(node_id, created_at, id);
