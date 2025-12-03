create table if not exists registries
(
    registry_id               SERIAL primary key,
    registry_name             text   not null
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
    registry_created_at       BIGINT not null,
    registry_updated_at       BIGINT not null,
    registry_created_by       INTEGER not null,
    registry_updated_by       INTEGER not null,
    registry_labels           text,
    constraint unique_registries
        unique (registry_root_parent_id, registry_name)
);


create table if not exists media_types
(
    mt_id         SERIAL primary key,
    mt_media_type text not null
        constraint unique_media_types_type
            unique,
    mt_created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM now()) * 1000)::BIGINT
);

create table if not exists blobs
(
    blob_id             SERIAL primary key,
    blob_root_parent_id INTEGER not null,
    blob_digest         bytea not null,
    blob_media_type_id  INTEGER not null
        constraint fk_blobs_media_type_id_media_types
            references media_types(mt_id),
    blob_size           BIGINT not null,
    blob_created_at     BIGINT not null,
    blob_created_by     INTEGER not null,
    constraint unique_digest_root_parent_id unique (blob_digest, blob_root_parent_id)
);

create index if not exists index_blobs_on_media_type_id
    on blobs (blob_media_type_id);

create table if not exists registry_blobs
(
    rblob_id          SERIAL primary key,
    rblob_registry_id INTEGER not null
        constraint fk_registry_blobs_rpstry_id_registries
            references registries
            on delete cascade,
    rblob_blob_id     INTEGER not null
        constraint fk_registry_blobs_blob_id_blobs
            references blobs
            on delete cascade,
    rblob_image_name  text
        constraint registry_blobs_image_len_check
            check (length(rblob_image_name) <= 255),
    rblob_created_at  BIGINT not null,
    rblob_updated_at  BIGINT not null,
    rblob_created_by  INTEGER not null,
    rblob_updated_by  INTEGER not null,

    constraint unique_registry_blobs_registry_id_blob_id_image
        unique (rblob_registry_id, rblob_blob_id, rblob_image_name)
);

create index if not exists index_registry_blobs_on_reg_id
    on registry_blobs (rblob_registry_id);

create index if not exists index_registry_blobs_on_reg_blob_id
    on registry_blobs (rblob_registry_id, rblob_blob_id);

create table if not exists manifests
(
    manifest_id                        SERIAL primary key,
    manifest_registry_id               INTEGER not null
        constraint fk_manifests_registry_id_registries
            references registries(registry_id)
            on delete cascade,
    manifest_schema_version            smallint not null,
    manifest_media_type_id             INTEGER not null
        constraint fk_manifests_media_type_id_media_types
            references media_types(mt_id),
    manifest_artifact_media_type       text,
    manifest_total_size                BIGINT not null,
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
    manifest_created_at                BIGINT not null,
    manifest_created_by                INTEGER not null,
    manifest_updated_at                BIGINT not null,
    manifest_updated_by                INTEGER not null,
    constraint unique_manifests_registry_id_image_name_and_digest
        unique (manifest_registry_id, manifest_image_name, manifest_digest),
    constraint unique_manifests_registry_id_id_cfg_blob_id
        unique (manifest_registry_id, manifest_id, manifest_configuration_blob_id),
    constraint fk_manifests_subject_id_manifests
        foreign key (manifest_subject_id) references manifests
            on delete cascade
);

create index if not exists index_manifests_on_media_type_id
    on manifests (manifest_media_type_id);

create index if not exists index_manifests_on_configuration_blob_id
    on manifests (manifest_configuration_blob_id);

create table if not exists manifest_references
(
    manifest_ref_id          SERIAL primary key,
    manifest_ref_registry_id INTEGER not null,
    manifest_ref_parent_id   INTEGER not null,
    manifest_ref_child_id    INTEGER not null,
    manifest_ref_created_at  BIGINT not null,
    manifest_ref_updated_at  BIGINT not null,
    manifest_ref_created_by  INTEGER not null,
    manifest_ref_updated_by  INTEGER not null,
    constraint unique_manifest_references_prt_id_chd_id
        unique (manifest_ref_registry_id, manifest_ref_parent_id, manifest_ref_child_id),
    constraint fk_manifest_references_parent_id_mnfsts
        foreign key (manifest_ref_parent_id) references manifests
            on delete cascade,
    constraint fk_manifest_references_child_id_mnfsts
        foreign key (manifest_ref_child_id) references manifests,
    constraint check_manifest_references_parent_id_and_child_id_differ
        check (manifest_ref_parent_id <> manifest_ref_child_id)
);

create index if not exists index_manifest_references_on_rpstry_id_child_id
    on manifest_references (manifest_ref_registry_id, manifest_ref_child_id);

create table if not exists layers
(
    layer_id            SERIAL primary key,
    layer_registry_id   INTEGER not null,
    layer_manifest_id   INTEGER not null,
    layer_media_type_id INTEGER not null
        constraint fk_layer_media_type_id_media_types
            references media_types,
    layer_blob_id       INTEGER not null
        constraint fk_layer_blob_id_blobs
            references blobs,
    layer_size          BIGINT not null,
    layer_created_at    BIGINT not null,
    layer_updated_at    BIGINT not null,
    layer_created_by    INTEGER not null,
    layer_updated_by    INTEGER not null,
    constraint unique_layer_rpstry_id_and_mnfst_id_and_blob_id
        unique (layer_registry_id, layer_manifest_id, layer_blob_id),
    constraint unique_layer_rpstry_id_and_id_and_blob_id
        unique (layer_registry_id, layer_id, layer_blob_id),
    constraint fk_manifst_id_manifests
        foreign key (layer_manifest_id) references manifests(manifest_id)
            on delete cascade
);

create index if not exists index_layer_on_media_type_id
    on layers (layer_media_type_id);

create index if not exists index_layer_on_blob_id
    on layers (layer_blob_id);

create table if not exists artifacts
(
    artifact_id                  SERIAL primary key,
    artifact_name                text not null,
    artifact_registry_id         INTEGER not null
    constraint fk_registries_registry_id
    references registries(registry_id)
    on delete cascade,
    artifact_labels              text,
    artifact_enabled             boolean default false,
    artifact_created_at          BIGINT,
    artifact_updated_at          BIGINT,
    artifact_created_by          INTEGER,
    artifact_updated_by          INTEGER,
    constraint unique_artifact_registry_id_and_name unique (artifact_registry_id, artifact_name),
    constraint check_artifact_name_length check ((char_length(artifact_name) <= 255))
);

create index if not exists index_artifact_on_registry_id ON artifacts USING btree (artifact_registry_id);


create table if not exists artifact_stats
(
    artifact_stat_id                               SERIAL primary key,
    artifact_stat_artifact_id                      INTEGER not null
    constraint fk_artifacts_artifact_id
    references artifacts(artifact_id),
    artifact_stat_date                             BIGINT,
    artifact_stat_download_count                   BIGINT,
    artifact_stat_upload_bytes                     BIGINT,
    artifact_stat_download_bytes                   BIGINT,
    artifact_stat_created_at                       BIGINT not null,
    artifact_stat_updated_at                       BIGINT not null,
    artifact_stat_created_by                       INTEGER not null,
    artifact_stat_updated_by                       INTEGER not null,
    constraint unique_artifact_stats_artifact_id_and_date unique (artifact_stat_artifact_id, artifact_stat_date)
);

create table if not exists tags
(
    tag_id          SERIAL primary key,
    tag_name        text not null
        constraint tag_name_len_check
            check (char_length(tag_name) <= 128),
    tag_image_name  text not null
        constraint tag_img_name_len_check
            check (length(tag_image_name) <= 255),
    tag_registry_id INTEGER not null,
    tag_manifest_id INTEGER not null,
    tag_created_at  BIGINT,
    tag_updated_at  BIGINT,
    tag_created_by  INTEGER,
    tag_updated_by  INTEGER,
    constraint fk_tag_manifest_id_manifests FOREIGN KEY
(tag_manifest_id) REFERENCES manifests (manifest_id) ON DELETE CASCADE,
    constraint unique_tag_registry_id_and_name_and_image_name
        unique (tag_registry_id, tag_name, tag_image_name)
);

create index if not exists index_tag_on_rpository_id_and_manifest_id
    on tags (tag_registry_id, tag_manifest_id);

create table if not exists upstream_proxy_configs
(
    upstream_proxy_config_id          SERIAL primary key,
    upstream_proxy_config_registry_id INTEGER not null
        constraint fk_upstream_proxy_config_registry_id
            references registries
            on delete cascade,
    upstream_proxy_config_source      text,
    upstream_proxy_config_url         text,
    upstream_proxy_config_auth_type   text not null,
    upstream_proxy_config_user_name   text,
    upstream_proxy_config_secret_identifier   text,
    upstream_proxy_config_secret_space_id    INTEGER,
        constraint fk_layers_secret_identifier_and_secret_space_id
        foreign key (upstream_proxy_config_secret_identifier, upstream_proxy_config_secret_space_id)
            references secrets(secret_uid, secret_space_id),
    upstream_proxy_config_token       text,
    upstream_proxy_config_created_at  BIGINT,
    upstream_proxy_config_updated_at  BIGINT,
    upstream_proxy_config_created_by  INTEGER,
    upstream_proxy_config_updated_by  INTEGER
);

create index if not exists index_upstream_proxy_config_on_registry_id
    on upstream_proxy_configs (upstream_proxy_config_registry_id);

create table if not exists cleanup_policies
(
    cp_id             SERIAL primary key,
    cp_registry_id    INTEGER not null
        constraint fk_cleanup_policies_registry_id
            references registries ON DELETE CASCADE,
    cp_name           text,
    cp_expiry_time_ms BIGINT,
    cp_created_at     BIGINT not null,
    cp_updated_at     BIGINT not null,
    cp_created_by     INTEGER not null,
    cp_updated_by     INTEGER not null
);

create index if not exists index_cleanup_policies_on_registry_id
    on cleanup_policies (cp_registry_id);

create table if not exists cleanup_policy_prefix_mappings
(
    cpp_id                SERIAL primary key,
    cpp_cleanup_policy_id INTEGER not null
        constraint fk_cleanup_policies_id
            references cleanup_policies(cp_id) ON DELETE CASCADE,
    cpp_prefix            text not null,
    cpp_prefix_type       text not null
);

create index if not exists index_cleanup_policy_map_on_policy_id
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
       ('application/vnd.cncf.helm.chart.provenance.v1.prov')
ON CONFLICT (mt_media_type)
DO NOTHING;

create table if not exists gc_blob_review_queue
(
    blob_id      INTEGER        NOT NULL,
    review_after BIGINT         NOT NULL DEFAULT (EXTRACT(EPOCH FROM (NOW() + INTERVAL '1 day'))),
    review_count INTEGER        NOT NULL DEFAULT 0,
    created_at   BIGINT         NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
    event        text           NOT NULL,
    CONSTRAINT pk_gc_blob_review_queue primary key (blob_id)
);

create index if not exists index_gc_blob_review_queue_on_review_after ON gc_blob_review_queue USING btree (review_after);

create table if not exists gc_review_after_defaults
(
    event text     NOT NULL,
    value interval NOT NULL,
    CONSTRAINT pk_gc_review_after_defaults PRIMARY KEY (event),
    CONSTRAINT check_gc_review_after_defaults_event_length CHECK ((char_length(event) <= 255))
);

INSERT INTO gc_review_after_defaults (event, value)
VALUES ('blob_upload', interval '1 day'),
       ('manifest_upload', interval '1 day'),
       ('manifest_delete', interval '1 day'),
       ('layer_delete', interval '1 day'),
       ('manifest_list_delete', interval '1 day'),
       ('tag_delete', interval '1 day'),
       ('tag_switch', interval '1 day')
ON CONFLICT (event)
    DO NOTHING;

create table if not exists gc_manifest_review_queue
(
    registry_id  INTEGER        NOT NULL,
    manifest_id  INTEGER        NOT NULL,
    review_after BIGINT         NOT NULL DEFAULT (EXTRACT(EPOCH FROM (NOW() + INTERVAL '1 day'))),
    review_count INTEGER        NOT NULL DEFAULT 0,
    created_at   BIGINT         NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
    event        text           NOT NULL,
    CONSTRAINT pk_gc_manifest_review_queue PRIMARY KEY (registry_id, manifest_id),
    CONSTRAINT fk_gc_manifest_review_queue_rp_id_mfst_id_mnfsts FOREIGN KEY (manifest_id) REFERENCES manifests (manifest_id) ON DELETE CASCADE
);

create index if not exists index_gc_manifest_review_queue_on_review_after ON gc_manifest_review_queue USING btree (review_after);

CREATE OR REPLACE FUNCTION gc_review_after(e text)
    RETURNS BIGINT
    VOLATILE
AS
$$
DECLARE
    result timestamp WITH time zone;
BEGIN
    SELECT (now() + value)
    INTO result
    FROM gc_review_after_defaults
    WHERE event = e;
    IF result IS NULL THEN
        RETURN EXTRACT(EPOCH FROM (now() + interval '1 day'));
    ELSE
        RETURN EXTRACT(EPOCH FROM result);
    END IF;
END;
$$
    LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION gc_track_blob_uploads()
    RETURNS TRIGGER
AS
$$
BEGIN
    INSERT INTO gc_blob_review_queue (blob_id, review_after, event)
    VALUES (NEW.blob_id, gc_review_after('blob_upload'), 'blob_upload')
    ON CONFLICT (blob_id)
        DO UPDATE SET review_after = gc_review_after('blob_upload'),
                      event        = 'blob_upload';
    RETURN NULL;
END;
$$
    LANGUAGE plpgsql;

CREATE TRIGGER gc_track_blob_uploads_trigger
    AFTER INSERT
    ON blobs
    FOR EACH ROW
EXECUTE PROCEDURE public.gc_track_blob_uploads();

CREATE OR REPLACE FUNCTION gc_track_manifest_uploads()
    RETURNS TRIGGER
AS
$$
BEGIN
    INSERT INTO gc_manifest_review_queue (registry_id, manifest_id, review_after, event)
    VALUES (NEW.manifest_registry_id, NEW.manifest_id, gc_review_after('manifest_upload'), 'manifest_upload');
    RETURN NULL;
END;
$$
    LANGUAGE plpgsql;

CREATE TRIGGER gc_track_manifest_uploads_trigger
    AFTER INSERT
    ON manifests
    FOR EACH ROW
EXECUTE PROCEDURE gc_track_manifest_uploads();

CREATE OR REPLACE FUNCTION gc_track_deleted_manifests()
    RETURNS TRIGGER
AS
$$
BEGIN
    IF OLD.manifest_configuration_blob_id IS NOT NULL THEN -- not all manifests have a configuration
INSERT INTO gc_blob_review_queue (blob_id, review_after, event)
VALUES (OLD.manifest_configuration_blob_id, gc_review_after('manifest_delete'), 'manifest_delete')
ON CONFLICT (blob_id)
    DO UPDATE SET
                  review_after = gc_review_after('manifest_delete'),
                  event = 'manifest_delete';
END IF;
RETURN NULL;
END;
$$
    LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION gc_track_deleted_layers()
    RETURNS TRIGGER
AS
$$
BEGIN
    IF (TG_LEVEL = 'STATEMENT') THEN
        INSERT INTO gc_blob_review_queue (blob_id, review_after, event)
        SELECT deleted_rows.layer_blob_id,
               gc_review_after('layer_delete'),
               'layer_delete'
        FROM old_table deleted_rows
                 JOIN
             blobs b ON deleted_rows.layer_blob_id = b.blob_id
        ORDER BY deleted_rows.layer_blob_id ASC
        ON CONFLICT (blob_id)
            DO UPDATE SET review_after = gc_review_after('layer_delete'),
                          event        = 'layer_delete';
    ELSIF (TG_LEVEL = 'ROW') THEN
        INSERT INTO gc_blob_review_queue (blob_id, review_after, event)
        VALUES (OLD.blob_id, gc_review_after('layer_delete'), 'layer_delete')
        ON CONFLICT (blob_id)
            DO UPDATE SET review_after = gc_review_after('layer_delete'),
                          event        = 'layer_delete';
    END IF;
    RETURN NULL;
END;
$$
    LANGUAGE plpgsql;

CREATE TRIGGER gc_track_deleted_manifests_trigger
    AFTER DELETE
    ON manifests
    FOR EACH ROW
EXECUTE PROCEDURE gc_track_deleted_manifests();

CREATE TRIGGER gc_track_deleted_layers_trigger
    AFTER DELETE
    ON layers
    REFERENCING OLD TABLE AS old_table
    FOR EACH STATEMENT
EXECUTE FUNCTION gc_track_deleted_layers();

CREATE OR REPLACE FUNCTION gc_track_deleted_manifest_lists()
    RETURNS TRIGGER
AS
$$
BEGIN
    INSERT INTO gc_manifest_review_queue (registry_id, manifest_id, review_after, event)
    VALUES (OLD.manifest_ref_registry_id, OLD.manifest_ref_child_id, gc_review_after('manifest_list_delete'), 'manifest_list_delete')
    ON CONFLICT (registry_id, manifest_id)
        DO UPDATE SET review_after = gc_review_after('manifest_list_delete'),
                      event        = 'manifest_list_delete';
    RETURN NULL;
END;
$$
    LANGUAGE plpgsql;

CREATE TRIGGER gc_track_deleted_manifest_lists_trigger
    AFTER DELETE
    ON manifest_references
    FOR EACH ROW
EXECUTE PROCEDURE gc_track_deleted_manifest_lists();


CREATE OR REPLACE FUNCTION gc_track_deleted_tags()
    RETURNS TRIGGER
AS
$$
BEGIN
    IF EXISTS (SELECT 1
               FROM manifests
               WHERE manifest_registry_id = OLD.tag_registry_id
                 AND manifest_id = OLD.tag_registry_id) THEN
        INSERT INTO gc_manifest_review_queue (registry_id, manifest_id, review_after, event)
        VALUES (OLD.tag_registry_id, OLD.tag_manifest_id, gc_review_after('tag_delete'), 'tag_delete')
        ON CONFLICT (registry_id, manifest_id)
            DO UPDATE SET review_after = gc_review_after('tag_delete'),
                          event        = 'tag_delete';
    END IF;
    RETURN NULL;
END;
$$
    LANGUAGE plpgsql;

CREATE TRIGGER gc_track_deleted_tag_trigger
    AFTER DELETE
    ON tags
    FOR EACH ROW
EXECUTE PROCEDURE gc_track_deleted_tags();

CREATE OR REPLACE FUNCTION gc_track_switched_tags()
    RETURNS TRIGGER
AS
$$
BEGIN
    INSERT INTO gc_manifest_review_queue (registry_id, manifest_id, review_after, event)
    VALUES (OLD.tag_registry_id, OLD.tag_manifest_id, gc_review_after('tag_switch'), 'tag_switch')
    ON CONFLICT (registry_id, manifest_id)
        DO UPDATE SET review_after = gc_review_after('tag_switch'),
                      event        = 'tag_switch';
    RETURN NULL;
END;
$$
    LANGUAGE plpgsql;

CREATE TRIGGER gc_track_switched_tag_trigger
    AFTER UPDATE OF tag_manifest_id
    ON tags
    FOR EACH ROW
EXECUTE PROCEDURE gc_track_switched_tags();
