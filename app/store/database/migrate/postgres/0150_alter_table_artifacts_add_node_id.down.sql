DROP INDEX IF EXISTS idx_artifacts_node_id;

ALTER TABLE artifacts DROP CONSTRAINT IF EXISTS fk_artifacts_node_id;

ALTER TABLE artifacts DROP COLUMN IF EXISTS artifact_node_id;