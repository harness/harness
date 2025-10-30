
ALTER TABLE nodes RENAME TO nodes_old;

CREATE TABLE nodes (
                       node_id              TEXT PRIMARY KEY,
                       node_name            TEXT    NOT NULL,
                       node_parent_id       TEXT,
                       node_registry_id     INTEGER NOT NULL REFERENCES registries (registry_id),
                       node_is_file         BOOLEAN NOT NULL,
                       node_path            TEXT    NOT NULL,
                       node_generic_blob_id TEXT REFERENCES generic_blobs (generic_blob_id),
                       node_created_at      INTEGER NOT NULL,
                       node_created_by      INTEGER NOT NULL,
                       CONSTRAINT unique_nodes UNIQUE (node_name, node_parent_id)
);

INSERT INTO nodes (
    node_id,
    node_name,
    node_parent_id,
    node_registry_id,
    node_is_file,
    node_path,
    node_generic_blob_id,
    node_created_at,
    node_created_by
)
SELECT
    node_id,
    node_name,
    node_parent_id,
    node_registry_id,
    node_is_file,
    node_path,
    node_generic_blob_id,
    node_created_at,
    node_created_by
FROM nodes_old;

DROP TABLE nodes_old;

CREATE TABLE artifacts_new (
                               artifact_id                  INTEGER PRIMARY KEY AUTOINCREMENT,
                               artifact_version             TEXT NOT NULL,
                               artifact_image_id            INTEGER NOT NULL
                                   CONSTRAINT fk_images_image_id
                                       REFERENCES images(image_id),
                               artifact_created_at          INTEGER,
                               artifact_updated_at          INTEGER,
                               artifact_created_by          INTEGER,
                               artifact_updated_by          INTEGER,
                               CONSTRAINT unique_artifact_image_id_and_version UNIQUE (artifact_image_id, artifact_version),
                               CONSTRAINT check_artifact_name_length CHECK ((length(artifact_version) <= 255))
);

INSERT INTO artifacts_new (
    artifact_id,
    artifact_version,
    artifact_image_id,
    artifact_created_at,
    artifact_updated_at,
    artifact_created_by,
    artifact_updated_by
)
SELECT
    artifact_id,
    artifact_version,
    artifact_image_id,
    artifact_created_at,
    artifact_updated_at,
    artifact_created_by,
    artifact_updated_by
FROM artifacts;

DROP TABLE artifacts;

ALTER TABLE artifacts_new RENAME TO artifacts;