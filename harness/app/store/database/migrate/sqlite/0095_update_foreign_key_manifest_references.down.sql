CREATE TABLE manifest_references_old
(
    manifest_ref_id          INTEGER PRIMARY KEY AUTOINCREMENT,
    manifest_ref_registry_id INTEGER NOT NULL,
    manifest_ref_parent_id   INTEGER NOT NULL,
    manifest_ref_child_id    INTEGER NOT NULL,
    manifest_ref_created_at  BIGINT  NOT NULL,
    manifest_ref_updated_at  BIGINT  NOT NULL,
    manifest_ref_created_by  INTEGER NOT NULL,
    manifest_ref_updated_by  INTEGER NOT NULL,
    CONSTRAINT unique_manifest_references_prt_id_chd_id
        UNIQUE (manifest_ref_registry_id, manifest_ref_parent_id, manifest_ref_child_id),
    CONSTRAINT fk_manifest_references_parent_id_mnfsts
        FOREIGN KEY (manifest_ref_parent_id) REFERENCES manifests
            ON DELETE CASCADE,
    CONSTRAINT fk_manifest_references_child_id_mnfsts
        FOREIGN KEY (manifest_ref_child_id) REFERENCES manifests,
    CONSTRAINT check_manifest_references_parent_id_and_child_id_differ
        CHECK (manifest_ref_parent_id <> manifest_ref_child_id)
);

INSERT INTO manifest_references_old (manifest_ref_id, manifest_ref_registry_id, manifest_ref_parent_id,
                                     manifest_ref_child_id, manifest_ref_created_at, manifest_ref_updated_at,
                                     manifest_ref_created_by, manifest_ref_updated_by)
SELECT manifest_ref_id,
       manifest_ref_registry_id,
       manifest_ref_parent_id,
       manifest_ref_child_id,
       manifest_ref_created_at,
       manifest_ref_updated_at,
       manifest_ref_created_by,
       manifest_ref_updated_by
FROM manifest_references;

DROP TABLE manifest_references;

ALTER TABLE manifest_references_old RENAME TO manifest_references;
