ALTER TABLE tasks ADD COLUMN current_stage_node_id TEXT;

ALTER TABLE nodes ADD COLUMN role TEXT;
ALTER TABLE nodes ADD COLUMN stage_node_id TEXT;
ALTER TABLE nodes ADD COLUMN active_run_id TEXT;
ALTER TABLE nodes ADD COLUMN last_run_at TEXT;
ALTER TABLE nodes ADD COLUMN run_count INTEGER NOT NULL DEFAULT 0;

UPDATE nodes
SET role = CASE
  WHEN role IS NOT NULL AND role != '' THEN role
  WHEN kind = 'group' THEN 'container'
  ELSE 'step'
END;

CREATE INDEX IF NOT EXISTS idx_nodes_role ON nodes(task_id, role) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_nodes_stage ON nodes(stage_node_id) WHERE deleted_at IS NULL;
