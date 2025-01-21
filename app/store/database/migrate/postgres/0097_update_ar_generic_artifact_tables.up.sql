CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
ALTER TABLE nodes DROP CONSTRAINT unique_nodes;
ALTER TABLE nodes ADD CONSTRAINT unique_nodes UNIQUE (node_registry_id, node_path);

ALTER TABLE artifacts
    ADD COLUMN artifact_metadata JSONB;

