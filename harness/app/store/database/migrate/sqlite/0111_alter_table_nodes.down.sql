ALTER TABLE nodes RENAME TO nodes_old;

CREATE TABLE nodes (
                       node_id              TEXT PRIMARY KEY,
                       node_name            TEXT    NOT NULL,
                       node_parent_id       TEXT REFERENCES nodes (node_id),
                       node_registry_id     INTEGER NOT NULL REFERENCES registries (registry_id),
                       node_is_file         BOOLEAN NOT NULL,
                       node_path            TEXT    NOT NULL,
                       node_generic_blob_id TEXT REFERENCES generic_blobs (generic_blob_id),
                       node_created_at      INTEGER NOT NULL,
                       node_created_by      INTEGER NOT NULL,
                       CONSTRAINT unique_nodes UNIQUE (node_registry_id, node_path)
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