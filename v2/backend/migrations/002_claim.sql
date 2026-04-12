-- 002: claim/lease fields on nodes
ALTER TABLE nodes ADD COLUMN claimed_by_type TEXT;
ALTER TABLE nodes ADD COLUMN claimed_by_id TEXT;
ALTER TABLE nodes ADD COLUMN lease_until TEXT;

CREATE INDEX IF NOT EXISTS idx_nodes_claim ON nodes(claimed_by_id, lease_until) WHERE deleted_at IS NULL;
