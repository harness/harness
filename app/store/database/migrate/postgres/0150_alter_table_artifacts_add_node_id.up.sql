ALTER TABLE artifacts ADD COLUMN artifact_node_id UUID;

ALTER TABLE artifacts ADD CONSTRAINT fk_artifacts_node_id
    FOREIGN KEY (artifact_node_id) REFERENCES nodes (node_id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_artifacts_node_id ON artifacts (artifact_node_id);