ALTER TABLE nodes DROP CONSTRAINT unique_nodes;
ALTER TABLE nodes ADD CONSTRAINT unique_node_path UNIQUE (node_registry_id, node_path);
ALTER TABLE artifacts DROP COLUMN artifact_metadata;