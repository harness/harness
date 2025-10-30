CREATE TABLE quarantined_paths_temp (
    quarantined_path_id TEXT PRIMARY KEY,
    quarantined_path_registry_id INTEGER NOT NULL,
    quarantined_path_image_id INTEGER REFERENCES images (image_id) ON DELETE CASCADE,
    quarantined_path_artifact_id INTEGER REFERENCES artifacts (artifact_id) ON DELETE CASCADE,
    quarantined_path_node_id TEXT REFERENCES nodes (node_id) ON DELETE CASCADE,
    quarantined_path_reason TEXT NOT NULL,
    quarantined_path_created_by INTEGER NOT NULL,
    quarantined_path_created_at INTEGER NOT NULL,
    FOREIGN KEY (quarantined_path_registry_id) REFERENCES registries (registry_id) ON DELETE CASCADE,
    UNIQUE (quarantined_path_registry_id, quarantined_path_image_id, quarantined_path_artifact_id, quarantined_path_node_id)
);

INSERT INTO quarantined_paths_temp SELECT * FROM quarantined_paths;
DROP TABLE quarantined_paths;
ALTER TABLE quarantined_paths_temp RENAME TO quarantined_paths;

CREATE INDEX IF NOT EXISTS idx_manifests_id_registry_image_name
    ON manifests (manifest_id, manifest_registry_id, manifest_image_name);

CREATE INDEX IF NOT EXISTS idx_tags_manifest_id_name
    ON tags (tag_manifest_id, tag_name);
