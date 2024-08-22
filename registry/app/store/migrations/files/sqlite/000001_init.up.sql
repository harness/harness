create table registries
(
    registry_id               INTEGER PRIMARY KEY AUTOINCREMENT,
    registry_name             text not null
        constraint registry_name_len_check
            check (length(registry_name) <= 255),
    registry_root_parent_id   INTEGER not null,
    registry_parent_id        INTEGER not null,
    registry_description      text,
    registry_type             text not null,
    registry_package_type     text not null,
    registry_upstream_proxies text,
    registry_allowed_pattern  text,
    registry_blocked_pattern  text,
    registry_labels           text,
    registry_created_at       INTEGER not null,
    registry_updated_at       INTEGER not null,
    registry_created_by       INTEGER not null,
    registry_updated_by       INTEGER not null,
    constraint unique_registries
        unique (registry_root_parent_id, registry_name)
);

create table media_types
(
    mt_id         INTEGER PRIMARY KEY AUTOINCREMENT,
    mt_media_type text not null
        constraint unique_media_types_type
            unique,
    mt_created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now') * 1000)
);

create table blobs
(
    blob_id             INTEGER PRIMARY KEY AUTOINCREMENT,
    blob_root_parent_id INTEGER not null,
    blob_digest         bytea not null,
    blob_media_type_id  INTEGER not null
    constraint fk_blobs_media_type_id_media_types
    references media_types(mt_id),
    blob_size           INTEGER not null,
    blob_created_at     INTEGER not null,
    blob_created_by     INTEGER not null,
    constraint unique_digest_root_parent_id unique (blob_digest, blob_root_parent_id)
    );

create index index_blobs_on_media_type_id
    on blobs (blob_media_type_id);

create table registry_blobs
(
    rblob_id          INTEGER PRIMARY KEY AUTOINCREMENT,
    rblob_registry_id INTEGER not null
    constraint fk_registry_blobs_rpstry_id_registries
    references registries(registry_id)
    on delete cascade,
    rblob_blob_id     INTEGER not null
        constraint fk_registry_blobs_blob_id_blobs
            references blobs(blob_id)
            on delete cascade,
    rblob_image_name  text
    constraint registry_blobs_image_len_check
    check (length(rblob_image_name) <= 255),
    rblob_created_at  INTEGER not null,
    rblob_updated_at  INTEGER not null,
    rblob_created_by  INTEGER not null,
    rblob_updated_by  INTEGER not null,

    constraint unique_registry_blobs_registry_id_blob_id_image
    unique (rblob_registry_id, rblob_blob_id, rblob_image_name)
    );

create index index_registry_blobs_on_reg_id
    on registry_blobs (rblob_registry_id);

create index index_registry_blobs_on_reg_blob_id
    on registry_blobs (rblob_registry_id, rblob_blob_id);



create table manifests
(
    manifest_id                        INTEGER PRIMARY KEY AUTOINCREMENT,
    manifest_registry_id               INTEGER not null
    constraint fk_manifests_registry_id_registries
    references registries(registry_id)
    on delete cascade,
    manifest_schema_version            smallint not null,
    manifest_media_type_id             INTEGER not null
    constraint fk_manifests_media_type_id_media_types
    references media_types(mt_id),
    manifest_artifact_media_type       text,
    manifest_total_size                INTEGER not null,
    manifest_configuration_media_type  text,
    manifest_configuration_payload     bytea,
    manifest_configuration_blob_id     INTEGER
    constraint fk_manifests_configuration_blob_id_blobs
    references blobs(blob_id),
    manifest_configuration_digest      bytea,
    manifest_digest                    bytea not null,
    manifest_payload                   bytea not null,
    manifest_non_conformant            boolean default false,
    manifest_non_distributable_layers  boolean default false,
    manifest_subject_id                INTEGER,
    manifest_subject_digest            bytea,
    manifest_annotations               bytea,
    manifest_image_name                text not null
    constraint manifests_img_name_len_check
    check (length(manifest_image_name) <= 255),
    manifest_created_at                INTEGER not null,
    manifest_created_by                INTEGER not null,
    manifest_updated_at                INTEGER not null,
    manifest_updated_by                INTEGER not null,
    constraint unique_manifests_registry_id_image_name_and_digest
    unique (manifest_registry_id, manifest_image_name, manifest_digest),
    constraint unique_manifests_registry_id_id_cfg_blob_id
    unique (manifest_registry_id, manifest_id, manifest_configuration_blob_id),
    constraint fk_manifests_subject_id_manifests
    foreign key (manifest_subject_id) references manifests(manifest_id)
    on delete cascade
    );

create index index_manifests_on_media_type_id
    on manifests (manifest_media_type_id);

create index index_manifests_on_configuration_blob_id
    on manifests (manifest_configuration_blob_id);



create table manifest_references
(
    manifest_ref_id          INTEGER PRIMARY KEY AUTOINCREMENT,
    manifest_ref_registry_id INTEGER not null,
    manifest_ref_parent_id   INTEGER not null,
    manifest_ref_child_id    INTEGER not null,
    manifest_ref_created_at  INTEGER not null,
    manifest_ref_updated_at  INTEGER not null,
    manifest_ref_created_by  INTEGER not null,
    manifest_ref_updated_by  INTEGER not null,
    constraint unique_manifest_references_prt_id_chd_id
    unique (manifest_ref_registry_id, manifest_ref_parent_id, manifest_ref_child_id),
    constraint fk_manifest_ref_parent_id_manifests_manifest_id
    foreign key (manifest_ref_parent_id) references manifests(manifest_id)
    on delete cascade,
    constraint fk_manifest_ref_child_id_manifests_manifest_id
    foreign key (manifest_ref_child_id) references manifests(manifest_id),
    constraint check_manifest_references_parent_id_and_child_id_differ
    check (manifest_ref_parent_id <> manifest_ref_child_id)
    );

create index index_manifest_references_on_rpstry_id_child_id
    on manifest_references (manifest_ref_registry_id, manifest_ref_child_id);

create table layers
(
    layer_id            INTEGER PRIMARY KEY AUTOINCREMENT,
    layer_registry_id   INTEGER not null,
    layer_manifest_id   INTEGER not null,
    layer_media_type_id INTEGER not null
    constraint fk_layer_media_type_id_media_types
    references media_types(mt_id),
    layer_blob_id       INTEGER not null
    constraint fk_layer_blob_id_blobs
    references blobs(blob_id),
    layer_size          INTEGER not null,
    layer_created_at    INTEGER not null,
    layer_updated_at    INTEGER not null,
    layer_created_by    INTEGER not null,
    layer_updated_by    INTEGER not null,
    constraint unique_layer_rpstry_id_and_mnfst_id_and_blob_id
    unique (layer_registry_id, layer_manifest_id, layer_blob_id),
    constraint unique_layer_rpstry_id_and_id_and_blob_id
    unique (layer_registry_id, layer_id, layer_blob_id),
    constraint fk_layer_manifest_id_and_manifests_manifest_id
    foreign key (layer_manifest_id) references manifests(manifest_id)
    on delete cascade
    );

create index index_layer_on_media_type_id
    on layers (layer_media_type_id);

create index index_layer_on_blob_id
    on layers (layer_blob_id);

create table artifacts
(
    artifact_id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    artifact_name                text not null,
    artifact_registry_id         INTEGER not null
    constraint fk_registries_registry_id
    references registries(registry_id)
    on delete cascade,
    artifact_labels              text,
    artifact_enabled             boolean default false,
    artifact_created_at          INTEGER,
    artifact_updated_at          INTEGER,
    artifact_created_by          INTEGER,
    artifact_updated_by          INTEGER,
    constraint unique_artifact_registry_id_and_name unique (artifact_registry_id, artifact_name),
    constraint check_artifact_name_length check ((length(artifact_name) <= 255))
    );

create index index_artifact_on_registry_id ON artifacts (artifact_registry_id);


create table artifact_stats
(
    artifact_stat_id                               INTEGER PRIMARY KEY AUTOINCREMENT,
    artifact_stat_artifact_id                      INTEGER not null
    constraint fk_artifacts_artifact_id
    references artifacts(artifact_id) on delete cascade,
    artifact_stat_date                             INTEGER,
    artifact_stat_download_count                   INTEGER,
    artifact_stat_upload_bytes                     INTEGER,
    artifact_stat_download_bytes                   INTEGER,
    artifact_stat_created_at                       INTEGER not null,
    artifact_stat_updated_at                       INTEGER not null,
    artifact_stat_created_by                       INTEGER not null,
    artifact_stat_updated_by                       INTEGER not null,
    constraint unique_artifact_stats_artifact_id_and_date unique (artifact_stat_artifact_id, artifact_stat_date)
    );

create table tags
(
    tag_id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tag_name        text not null
    constraint tag_name_len_check
    check (length(tag_name) <= 128),
    tag_image_name  text not null
    constraint tag_img_name_len_check
    check (length(tag_image_name) <= 255),
    tag_registry_id INTEGER not null,
    tag_manifest_id INTEGER not null,
    tag_created_at  INTEGER,
    tag_updated_at  INTEGER,
    tag_created_by  INTEGER,
    tag_updated_by  INTEGER,
    constraint fk_tag_manifest_id_and_manifests_manifest_id FOREIGN KEY
     (tag_manifest_id) REFERENCES manifests (manifest_id) ON DELETE CASCADE,
    constraint unique_tag_registry_id_and_name_and_image_name
    unique (tag_registry_id, tag_name, tag_image_name)
    );

create index index_tag_on_rpository_id_and_manifest_id
    on tags (tag_registry_id, tag_manifest_id);

create table upstream_proxy_configs
(
    upstream_proxy_config_id          INTEGER PRIMARY KEY AUTOINCREMENT,
    upstream_proxy_config_registry_id INTEGER not null
    constraint fk_upstream_proxy_config_registry_id
    references registries(registry_id)
    on delete cascade,
    upstream_proxy_config_source      text,
    upstream_proxy_config_url         text,
    upstream_proxy_config_auth_type   text not null,
    upstream_proxy_config_user_name   text,
    upstream_proxy_config_secret_identifier    text,
    upstream_proxy_config_secret_space_id int,
    upstream_proxy_config_token       text,
    upstream_proxy_config_created_at  INTEGER,
    upstream_proxy_config_updated_at  INTEGER,
    upstream_proxy_config_created_by  INTEGER,
    upstream_proxy_config_updated_by  INTEGER,
    constraint fk_layers_secret_identifier_and_secret_space_id FOREIGN KEY
     (upstream_proxy_config_secret_identifier, upstream_proxy_config_secret_space_id) REFERENCES secrets(secret_uid, secret_space_id)
        ON DELETE CASCADE
);

create index index_upstream_proxy_config_on_registry_id
    on upstream_proxy_configs (upstream_proxy_config_registry_id);

create table cleanup_policies
(
    cp_id             INTEGER PRIMARY KEY AUTOINCREMENT,
    cp_registry_id    INTEGER not null
    constraint fk_cleanup_policies_registry_id
    references registries(registry_id) ON DELETE CASCADE,
    cp_name           text,
    cp_expiry_time_ms INTEGER,
    cp_created_at     INTEGER not null,
    cp_updated_at     INTEGER not null,
    cp_created_by     INTEGER not null,
    cp_updated_by     INTEGER not null
);

create index index_cleanup_policies_on_registry_id
    on cleanup_policies (cp_registry_id);

create table cleanup_policy_prefix_mappings
(
    cpp_id                INTEGER PRIMARY KEY AUTOINCREMENT,
    cpp_cleanup_policy_id INTEGER not null
    constraint fk_cleanup_policy_prefix_registry_id
    references cleanup_policies(cp_id) ON DELETE CASCADE,
    cpp_prefix            text not null,
    cpp_prefix_type       text not null
);

create index index_cleanup_policy_map_on_policy_id
    on cleanup_policy_prefix_mappings (cpp_cleanup_policy_id);



insert into media_types (mt_media_type)
values ('application/vnd.docker.distribution.manifest.v1+json'),
       ('application/vnd.docker.distribution.manifest.v1+prettyjws'),
       ('application/vnd.docker.distribution.manifest.v2+json'),
       ('application/vnd.docker.distribution.manifest.list.v2+json'),
       ('application/vnd.docker.image.rootfs.diff.tar'),
       ('application/vnd.docker.image.rootfs.diff.tar.gzip'),
       ('application/vnd.docker.image.rootfs.foreign.diff.tar.gzip'),
       ('application/vnd.docker.container.image.v1+json'),
       ('application/vnd.docker.container.image.rootfs.diff+x-gtar'),
       ('application/vnd.docker.plugin.v1+json'),
       ('application/vnd.oci.image.layer.v1.tar'),
       ('application/vnd.oci.image.layer.v1.tar+gzip'),
       ('application/vnd.oci.image.layer.v1.tar+zstd'),
       ('application/vnd.oci.image.layer.nondistributable.v1.tar'),
       ('application/vnd.oci.image.layer.nondistributable.v1.tar+gzip'),
       ('application/vnd.oci.image.config.v1+json'),
       ('application/vnd.oci.image.manifest.v1+json'),
       ('application/vnd.oci.image.index.v1+json'),
       ('application/vnd.cncf.helm.config.v1+json'),
       ('application/tar+gzip'),
       ('application/octet-stream'),
       ('application/vnd.buildkit.cacheconfig.v0'),
       ('application/vnd.cncf.helm.chart.content.v1.tar+gzip'),
       ('application/vnd.cncf.helm.chart.provenance.v1.prov');
