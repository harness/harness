create table if not exists nodes
(
    node_id               TEXT PRIMARY KEY,
    node_name             text not null,
    node_parent_id        INTEGER not null,
    node_registry_id      INTEGER not null
    CONSTRAINT fk_registry_registry_id
    REFERENCES registries (registry_id) ,
    node_created_at       BIGINT not null,
    node_created_by       INTEGER not null,
    node_is_file          BOOLEAN not null,
    node_path    text not null,
    constraint unique_nodes
    unique (node_name, node_parent_id, node_registry_id)
    );



create table if not exists generic_blob
(
    generic_blob_id           TEXT PRIMARY KEY,
    generic_blob_root_parent_id  INTEGER not null ,
    generic_blob_sha_1         TEXT not null,
    generic_blob_sha_256       TEXT not null,
    generic_blob_sha_512       TEXT not null,
    generic_blob_MD5           TEXT not null,
    generic_blob_size          INTEGER not null,
    generic_blob_created_at       BIGINT not null,
    generic_blob_created_by       INTEGER not null,
    constraint unique_hash_root_parent_id unique (generic_blob_sha_256, generic_blob_root_parent_id)
    );


create table if not exists file_nodes
(
    file_node_id              TEXT PRIMARY KEY,
    file_node_name             text not null,
    file_node_parent_node_id        INTEGER not null
    CONSTRAINT fk_node_node_id
    REFERENCES nodes(node_id) ,
    file_node_generic_blob_id      INTEGER not null
    CONSTRAINT fk_generic_blob_generic_blob_id
    REFERENCES generic_blob(generic_blob_id) ,
    file_node_created_at       BIGINT not null,
    file_node_created_by       INTEGER not null,
    constraint unique_file_node
    unique (file_node_parent_node_id, file_node_name)
    );


