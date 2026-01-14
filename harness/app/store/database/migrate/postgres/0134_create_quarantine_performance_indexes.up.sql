ALTER TABLE quarantined_paths DROP CONSTRAINT IF EXISTS uq_quarantined_paths_composite;

ALTER TABLE quarantined_paths ADD CONSTRAINT uq_quarantined_paths_composite 
    UNIQUE (quarantined_path_registry_id, quarantined_path_image_id, quarantined_path_artifact_id, quarantined_path_node_id);

CREATE INDEX IF NOT EXISTS idx_manifests_id_registry_image_name
    ON manifests (manifest_id, manifest_registry_id, manifest_image_name);

CREATE INDEX IF NOT EXISTS idx_tags_manifest_id_image_name
    ON tags (tag_manifest_id, tag_name);

