-- schema_version 001: initial
CREATE TABLE IF NOT EXISTS schema_version (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  task_key TEXT,
  title TEXT NOT NULL,
  goal TEXT,
  status TEXT NOT NULL DEFAULT 'running',
  source_tool TEXT,
  source_session_id TEXT,
  created_by_type TEXT,
  created_by_id TEXT,
  creation_mode TEXT,
  creation_reason TEXT,
  tags_json TEXT NOT NULL DEFAULT '[]',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  summary_percent REAL NOT NULL DEFAULT 0,
  summary_done INTEGER NOT NULL DEFAULT 0,
  summary_total INTEGER NOT NULL DEFAULT 0,
  summary_blocked INTEGER NOT NULL DEFAULT 0,
  deleted_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_updated ON tasks(updated_at);

CREATE TABLE IF NOT EXISTS nodes (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL REFERENCES tasks(id),
  parent_node_id TEXT REFERENCES nodes(id),
  node_key TEXT,
  path TEXT NOT NULL,
  depth INTEGER NOT NULL,
  kind TEXT NOT NULL DEFAULT 'leaf',           -- leaf / group / linked_task
  linked_task_id TEXT REFERENCES tasks(id),
  title TEXT NOT NULL,
  instruction TEXT,
  acceptance_criteria_json TEXT NOT NULL DEFAULT '[]',
  status TEXT NOT NULL DEFAULT 'ready',
  progress REAL NOT NULL DEFAULT 0,
  estimate REAL NOT NULL DEFAULT 1,
  sort_order INTEGER NOT NULL DEFAULT 0,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  created_by_type TEXT,
  created_by_id TEXT,
  creation_reason TEXT,
  deleted_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_nodes_task ON nodes(task_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_nodes_parent ON nodes(parent_node_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_nodes_linked ON nodes(linked_task_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS events (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL,
  node_id TEXT,
  type TEXT NOT NULL,
  message TEXT,
  payload_json TEXT NOT NULL DEFAULT '{}',
  actor_type TEXT,
  actor_id TEXT,
  idempotency_key TEXT,
  created_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_events_idem ON events(idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_events_task ON events(task_id, created_at);
CREATE INDEX IF NOT EXISTS idx_events_node ON events(node_id, created_at);

CREATE TABLE IF NOT EXISTS artifacts (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL,
  node_id TEXT,
  kind TEXT,
  title TEXT,
  uri TEXT,
  meta_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_artifacts_task ON artifacts(task_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_node ON artifacts(node_id);

CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value_json TEXT NOT NULL
);
