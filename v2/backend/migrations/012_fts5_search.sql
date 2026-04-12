-- schema_version 012: FTS5 full-text search index
CREATE VIRTUAL TABLE IF NOT EXISTS search_index USING fts5(
  entity_type,
  entity_id,
  task_id,
  title,
  content,
  tokenize='unicode61'
);
