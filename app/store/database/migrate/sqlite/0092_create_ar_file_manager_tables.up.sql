create table if not exists generic_blobs
(
    generic_blob_id             TEXT PRIMARY KEY,
    generic_blob_root_parent_id INTEGER NOT NULL,
    generic_blob_sha_1          BLOB,
    generic_blob_sha_256        BLOB,
    generic_blob_sha_512        BLOB    NOT NULL,
    generic_blob_md5            BLOB,
    generic_blob_size           INTEGER NOT NULL,
    generic_blob_created_at     INTEGER NOT NULL,
    generic_blob_created_by     INTEGER NOT NULL,
    CONSTRAINT unique_generic_blob_parent_id unique (generic_blob_sha_256, generic_blob_root_parent_id)
);

create table if not exists nodes
(
    node_id              TEXT PRIMARY KEY,
    node_name            TEXT    NOT NULL,
    node_parent_id       INTEGER REFERENCES nodes (node_id),
    node_registry_id     INTEGER NOT NULL REFERENCES registries (registry_id),
    node_is_file         BOOLEAN NOT NULL,
    node_path            TEXT    NOT NULL,
    node_generic_blob_id TEXT REFERENCES generic_blobs (generic_blob_id),
    node_created_at      INTEGER NOT NULL,
    node_created_by      INTEGER NOT NULL,
    constraint unique_nodes
        unique (node_name, node_parent_id)
);

CREATE INDEX IF NOT EXISTS idx_nodes_parent_name ON nodes (node_parent_id, node_name);
CREATE INDEX IF NOT EXISTS idx_nodes_registry ON nodes (node_registry_id);
CREATE INDEX IF NOT EXISTS idx_nodes_blob ON nodes (node_generic_blob_id);
CREATE INDEX IF NOT EXISTS idx_generic_blobs_sha256 ON generic_blobs (generic_blob_sha_256);

