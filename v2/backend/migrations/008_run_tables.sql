CREATE TABLE IF NOT EXISTS node_runs (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL REFERENCES tasks(id),
  node_id TEXT NOT NULL REFERENCES nodes(id),
  stage_node_id TEXT REFERENCES nodes(id),
  actor_type TEXT,
  actor_id TEXT,
  trigger_kind TEXT NOT NULL DEFAULT 'manual',
  status TEXT NOT NULL DEFAULT 'running',
  result TEXT,
  input_summary TEXT,
  output_preview TEXT,
  output_ref TEXT,
  structured_result_json TEXT NOT NULL DEFAULT '{}',
  error_text TEXT,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_node_runs_task ON node_runs(task_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_node_runs_node ON node_runs(node_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_node_runs_stage ON node_runs(stage_node_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_node_runs_status ON node_runs(status, created_at DESC);

CREATE TABLE IF NOT EXISTS run_logs (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL REFERENCES node_runs(id) ON DELETE CASCADE,
  seq INTEGER NOT NULL,
  kind TEXT NOT NULL,
  content TEXT,
  payload_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  UNIQUE(run_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_run_logs_run ON run_logs(run_id, seq);
