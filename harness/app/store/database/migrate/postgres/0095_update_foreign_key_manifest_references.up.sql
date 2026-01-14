ALTER TABLE manifest_references
DROP
CONSTRAINT IF EXISTS fk_manifest_references_child_id_mnfsts;

ALTER TABLE manifest_references
    ADD CONSTRAINT fk_manifest_references_child_id_mnfsts
        FOREIGN KEY (manifest_ref_child_id) REFERENCES manifests (manifest_id)
            ON DELETE CASCADE;