CREATE TABLE images
(
    image_id                  SERIAL PRIMARY KEY,
    image_name                TEXT NOT NULL,
    image_registry_id         INTEGER NOT NULL
        CONSTRAINT fk_registries_registry_id
            REFERENCES registries(registry_id),
    image_labels              text,
    image_enabled             BOOLEAN DEFAULT FALSE,
    image_created_at          BIGINT NOT NULL,
    image_updated_at          BIGINT NOT NULL,
    image_created_by          INTEGER NOT NULL,
    image_updated_by          INTEGER NOT NULL,
    CONSTRAINT unique_image_registry_id_and_name UNIQUE (image_registry_id, image_name),
    CONSTRAINT check_image_name_length CHECK ((LENGTH(image_name) <= 255))
);

INSERT INTO images (image_name, image_registry_id, image_labels, image_enabled, image_created_at,
                    image_updated_at, image_created_by, image_updated_by)
SELECT artifact_name AS image_name,
       artifact_registry_id AS image_registry_id,
       artifact_labels AS image_labels,
       artifact_enabled AS image_enabled,
       artifact_created_at AS image_created_at,
       artifact_updated_at AS image_updated_at,
       artifact_created_by AS image_created_by,
       artifact_updated_by AS image_updated_by
FROM artifacts;


CREATE TABLE artifacts_temp
(
    artifact_id                  SERIAL PRIMARY KEY,
    artifact_version             TEXT NOT NULL,
    artifact_image_id            INTEGER NOT NULL
        CONSTRAINT fk_images_image_id
            REFERENCES images(image_id),
    artifact_created_at          BIGINT NOT NULL,
    artifact_updated_at          BIGINT NOT NULL,
    artifact_created_by          INTEGER NOT NULL,
    artifact_updated_by          INTEGER NOT NULL,
    CONSTRAINT unique_artifact_image_id_and_version UNIQUE (artifact_image_id, artifact_version)
);

INSERT INTO artifacts_temp (artifact_version, artifact_image_id, artifact_created_at, artifact_updated_at,
                            artifact_created_by, artifact_updated_by)
SELECT encode(m.manifest_digest, 'hex') AS artifact_version,
       i.image_id AS artifact_image_id,
       m.manifest_created_at AS artifact_created_at,
       m.manifest_updated_at AS artifact_updated_at,
       m.manifest_created_by AS artifact_created_by,
       m.manifest_updated_by AS artifact_updated_by
FROM artifacts a
         JOIN images i ON a.artifact_name = i.image_name AND a.artifact_registry_id = i.image_registry_id
         JOIN manifests m ON a.artifact_name = m.manifest_image_name AND a.artifact_registry_id = m.manifest_registry_id;


DROP INDEX index_artifact_on_registry_id;

CREATE TABLE temp_artifact_stats AS
SELECT *
FROM artifact_stats;

DROP TABLE artifact_stats;

DROP TABLE artifacts;

ALTER TABLE artifacts_temp
    RENAME TO artifacts;

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

INSERT INTO artifact_stats (
    artifact_stat_id,
    artifact_stat_artifact_id,
    artifact_stat_date,
    artifact_stat_download_count,
    artifact_stat_upload_bytes,
    artifact_stat_download_bytes,
    artifact_stat_created_at,
    artifact_stat_updated_at,
    artifact_stat_created_by,
    artifact_stat_updated_by
)
SELECT
    artifact_stat_id,
    artifact_stat_artifact_id,
    artifact_stat_date,
    artifact_stat_download_count,
    artifact_stat_upload_bytes,
    artifact_stat_download_bytes,
    artifact_stat_created_at,
    artifact_stat_updated_at,
    artifact_stat_created_by,
    artifact_stat_updated_by
FROM temp_artifact_stats;

DROP TABLE temp_artifact_stats;