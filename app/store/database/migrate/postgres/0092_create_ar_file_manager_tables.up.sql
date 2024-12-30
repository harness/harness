CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS generic_blobs
(
    generic_blob_id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    generic_blob_root_parent_id INTEGER NOT NULL,
    generic_blob_sha_1          BYTEA,
    generic_blob_sha_256        BYTEA   NOT NULL,
    generic_blob_sha_512        BYTEA,
    generic_blob_md5            BYTEA,
    generic_blob_size           INTEGER NOT NULL,
    generic_blob_created_at     BIGINT  NOT NULL,
    generic_blob_created_by     INTEGER NOT NULL,
    CONSTRAINT unique_generic_blobs_sha_parent
        UNIQUE (generic_blob_sha_256, generic_blob_root_parent_id)
);


CREATE TABLE IF NOT EXISTS nodes
(
    node_id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_name            TEXT    NOT NULL,
    node_parent_id       UUID REFERENCES nodes (node_id),
    node_registry_id     INTEGER NOT NULL REFERENCES registries (registry_id),
    node_is_file         BOOLEAN NOT NULL,
    node_path            TEXT    NOT NULL,
    node_generic_blob_id UUID REFERENCES generic_blobs (generic_blob_id),
    node_created_at      BIGINT  NOT NULL,
    node_created_by      INTEGER NOT NULL,
    CONSTRAINT unique_nodes
        UNIQUE (node_name, node_parent_id)
);


CREATE INDEX IF NOT EXISTS idx_nodes_parent_name ON nodes (node_parent_id, node_name);
CREATE INDEX IF NOT EXISTS idx_nodes_registry ON nodes (node_registry_id);
CREATE INDEX IF NOT EXISTS idx_nodes_blob ON nodes (node_generic_blob_id);
CREATE INDEX IF NOT EXISTS idx_generic_blobs_sha256 ON generic_blobs (generic_blob_sha_256);