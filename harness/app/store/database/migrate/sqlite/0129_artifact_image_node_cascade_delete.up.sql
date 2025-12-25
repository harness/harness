create table artifacts_dg_tmp
(
    artifact_id         INTEGER
        primary key autoincrement,
    artifact_version    TEXT    not null,
    artifact_image_id   INTEGER not null
        constraint fk_images_image_id
            references images
            on delete cascade,
    artifact_created_at INTEGER not null,
    artifact_updated_at INTEGER not null,
    artifact_created_by INTEGER not null,
    artifact_updated_by INTEGER not null,
    artifact_metadata   TEXT,
    constraint unique_artifact_image_id_and_version
        unique (artifact_image_id, artifact_version)
);

insert into artifacts_dg_tmp(artifact_id, artifact_version, artifact_image_id, artifact_created_at, artifact_updated_at,
                             artifact_created_by, artifact_updated_by, artifact_metadata)
select artifact_id,
       artifact_version,
       artifact_image_id,
       artifact_created_at,
       artifact_updated_at,
       artifact_created_by,
       artifact_updated_by,
       artifact_metadata
from artifacts;

drop table artifacts;

alter table artifacts_dg_tmp
    rename to artifacts;

---------------------


create table images_dg_tmp
(
    image_id          INTEGER
        primary key autoincrement,
    image_name        TEXT    not null,
    image_registry_id INTEGER not null
        constraint fk_registries_registry_id
            references registries
            on delete cascade,
    image_labels      text,
    image_enabled     BOOLEAN default FALSE,
    image_created_at  INTEGER not null,
    image_updated_at  INTEGER not null,
    image_created_by  INTEGER not null,
    image_updated_by  INTEGER not null,
    constraint unique_image_registry_id_and_name
        unique (image_registry_id, image_name),
    constraint check_image_name_length
        check ((LENGTH(image_name) <= 255))
);

insert into images_dg_tmp(image_id, image_name, image_registry_id, image_labels, image_enabled, image_created_at,
                          image_updated_at, image_created_by, image_updated_by)
select image_id,
       image_name,
       image_registry_id,
       image_labels,
       image_enabled,
       image_created_at,
       image_updated_at,
       image_created_by,
       image_updated_by
from images;

drop table images;

alter table images_dg_tmp
    rename to images;


---------------------------------------------------------------------------------------------------------


create table nodes_dg_tmp
(
    node_id              TEXT
        primary key,
    node_name            TEXT    not null,
    node_parent_id       TEXT
        references nodes
            on delete cascade,
    node_registry_id     INTEGER not null
        references registries
            on delete cascade,
    node_is_file         BOOLEAN not null,
    node_path            TEXT    not null,
    node_generic_blob_id TEXT
        references generic_blobs,
    node_created_at      INTEGER not null,
    node_created_by      INTEGER not null,
    constraint unique_nodes
        unique (node_registry_id, node_path)
);

insert into nodes_dg_tmp(node_id, node_name, node_parent_id, node_registry_id, node_is_file, node_path,
                         node_generic_blob_id, node_created_at, node_created_by)
select node_id,
       node_name,
       node_parent_id,
       node_registry_id,
       node_is_file,
       node_path,
       node_generic_blob_id,
       node_created_at,
       node_created_by
from nodes;

drop table nodes;

alter table nodes_dg_tmp
    rename to nodes;

create index if not exists layer_manifest_id_idx
    on layers (layer_manifest_id);

create index if not exists manifest_manifest_subject_id_idx
    on manifests (manifest_subject_id);

create index if not exists  tag_manifest_id_index
    on tags (tag_manifest_id);

create index if not exists  manifest_ref_child_id_idx
    on manifest_references (manifest_ref_child_id);

create index if not exists  manifest_ref_parent_id_idx
    on manifest_references (manifest_ref_parent_id);