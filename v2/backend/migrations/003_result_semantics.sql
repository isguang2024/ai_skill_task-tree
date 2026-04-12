-- 003: result semantics on tasks/nodes
ALTER TABLE tasks ADD COLUMN result TEXT NOT NULL DEFAULT '';
ALTER TABLE nodes ADD COLUMN result TEXT NOT NULL DEFAULT '';

UPDATE tasks
SET result = CASE
  WHEN status IN ('done', 'canceled') THEN status
  ELSE ''
END
WHERE COALESCE(result, '') = '';

UPDATE nodes
SET result = CASE
  WHEN status IN ('done', 'canceled') THEN status
  ELSE ''
END
WHERE COALESCE(result, '') = '';

UPDATE nodes
SET progress = 0
WHERE result = 'canceled' AND progress >= 1;
