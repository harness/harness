CREATE TABLE artifacts_temp
(
    artifact_id                  SERIAL PRIMARY KEY,
    artifact_name                TEXT NOT NULL,
    artifact_registry_id         INTEGER NOT NULL
        CONSTRAINT fk_registries_registry_id
            REFERENCES registries(registry_id)
            ON DELETE CASCADE,
    artifact_labels              TEXT,
    artifact_enabled             BOOLEAN DEFAULT FALSE,
    artifact_created_at          BIGINT,
    artifact_updated_at          BIGINT,
    artifact_created_by          INTEGER,
    artifact_updated_by          INTEGER,
    CONSTRAINT unique_artifact_registry_id_and_name UNIQUE (artifact_registry_id, artifact_name),
    CONSTRAINT check_artifact_name_length CHECK ((LENGTH(artifact_name) <= 255))
);

INSERT INTO artifacts_temp (artifact_id, artifact_name, artifact_registry_id, artifact_labels, artifact_enabled,
                            artifact_created_at, artifact_updated_at, artifact_created_by, artifact_updated_by)
SELECT i.image_id AS artifact_id,
       i.image_name AS artifact_name,
       i.image_registry_id AS artifact_registry_id,
       i.image_labels AS artifact_labels,
       i.image_enabled AS artifact_enabled,
       i.image_created_at AS artifact_created_at,
       i.image_updated_at AS artifact_updated_at,
       i.image_created_by AS artifact_created_by,
       i.image_updated_by AS artifact_updated_by
FROM images i;

DROP TABLE artifacts;

ALTER TABLE artifacts_temp
    RENAME TO artifacts;

create table if not exists artifact_stats
(
    artifact_stat_id                               SERIAL PRIMARY KEY,
    artifact_stat_artifact_id                      INTEGER NOT NULL
    CONSTRAINT fk_artifacts_artifact_id
    REFERENCES artifacts(artifact_id),
    artifact_stat_date                             BIGINT,
    artifact_stat_download_count                   BIGINT,
    artifact_stat_upload_bytes                     BIGINT,
    artifact_stat_download_bytes                   BIGINT,
    artifact_stat_created_at                       BIGINT NOT NULL,
    artifact_stat_updated_at                       BIGINT NOT NULL,
    artifact_stat_created_by                       INTEGER NOT NULL,
    artifact_stat_updated_by                       INTEGER NOT NULL,
    CONSTRAINT unique_artifact_stats_artifact_id_and_date UNIQUE (artifact_stat_artifact_id, artifact_stat_date)
    );

INSERT INTO artifact_stats (artifact_stat_artifact_id, artifact_stat_date, artifact_stat_download_count,
                            artifact_stat_upload_bytes, artifact_stat_download_bytes, artifact_stat_created_at,
                            artifact_stat_updated_at, artifact_stat_created_by, artifact_stat_updated_by)
SELECT a.artifact_id AS artifact_stat_artifact_id,
        (EXTRACT(EPOCH FROM now()) * 1000)::BIGINT AS artifact_stat_date,
        0 AS artifact_stat_download_count,
        0 AS artifact_stat_upload_bytes,
        0 AS artifact_stat_download_bytes,
        (EXTRACT(EPOCH FROM now()) * 1000)::BIGINT AS artifact_stat_created_at,
        (EXTRACT(EPOCH FROM now()) * 1000)::BIGINT AS artifact_stat_updated_at,
        a.artifact_created_by AS artifact_stat_created_by,
        a.artifact_updated_by AS artifact_stat_updated_by
FROM artifacts a;

CREATE INDEX index_artifact_on_registry_id ON artifacts (artifact_registry_id);

DROP TABLE images;
