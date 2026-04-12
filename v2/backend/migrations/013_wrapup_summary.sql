-- 013: add wrapup_summary to tasks
ALTER TABLE tasks ADD COLUMN wrapup_summary TEXT NOT NULL DEFAULT '';
