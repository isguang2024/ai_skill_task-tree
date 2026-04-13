-- schema_version 014: task context fields for onboarding and review snapshots
ALTER TABLE task_memory_current ADD COLUMN architecture_decisions_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE task_memory_current ADD COLUMN reference_files_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE task_memory_current ADD COLUMN context_doc_text TEXT NOT NULL DEFAULT '';
