ALTER TABLE artifacts ADD COLUMN run_id TEXT REFERENCES node_runs(id);

CREATE INDEX IF NOT EXISTS idx_artifacts_run ON artifacts(run_id);

CREATE TABLE IF NOT EXISTS node_memory_current (
  node_id TEXT PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  stage_node_id TEXT REFERENCES nodes(id),
  summary_text TEXT NOT NULL DEFAULT '',
  conclusions_json TEXT NOT NULL DEFAULT '[]',
  decisions_json TEXT NOT NULL DEFAULT '[]',
  risks_json TEXT NOT NULL DEFAULT '[]',
  blockers_json TEXT NOT NULL DEFAULT '[]',
  next_actions_json TEXT NOT NULL DEFAULT '[]',
  evidence_json TEXT NOT NULL DEFAULT '[]',
  manual_note_text TEXT NOT NULL DEFAULT '',
  source_run_id TEXT REFERENCES node_runs(id),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_node_memory_task ON node_memory_current(task_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_node_memory_stage ON node_memory_current(stage_node_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS stage_memory_current (
  stage_node_id TEXT PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  summary_text TEXT NOT NULL DEFAULT '',
  conclusions_json TEXT NOT NULL DEFAULT '[]',
  decisions_json TEXT NOT NULL DEFAULT '[]',
  risks_json TEXT NOT NULL DEFAULT '[]',
  blockers_json TEXT NOT NULL DEFAULT '[]',
  next_actions_json TEXT NOT NULL DEFAULT '[]',
  evidence_json TEXT NOT NULL DEFAULT '[]',
  manual_note_text TEXT NOT NULL DEFAULT '',
  source_snapshot_ref TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_stage_memory_task ON stage_memory_current(task_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS task_memory_current (
  task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
  current_stage_node_id TEXT REFERENCES nodes(id),
  summary_text TEXT NOT NULL DEFAULT '',
  conclusions_json TEXT NOT NULL DEFAULT '[]',
  decisions_json TEXT NOT NULL DEFAULT '[]',
  risks_json TEXT NOT NULL DEFAULT '[]',
  blockers_json TEXT NOT NULL DEFAULT '[]',
  next_actions_json TEXT NOT NULL DEFAULT '[]',
  evidence_json TEXT NOT NULL DEFAULT '[]',
  manual_note_text TEXT NOT NULL DEFAULT '',
  source_snapshot_ref TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS memory_snapshots (
  id TEXT PRIMARY KEY,
  scope_kind TEXT NOT NULL,
  task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  stage_node_id TEXT REFERENCES nodes(id),
  node_id TEXT REFERENCES nodes(id),
  summary_text TEXT NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL DEFAULT '{}',
  reason TEXT,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_memory_snapshots_task ON memory_snapshots(task_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_snapshots_stage ON memory_snapshots(stage_node_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_snapshots_node ON memory_snapshots(node_id, created_at DESC);
