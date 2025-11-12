CREATE TABLE artifacts_original (
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
    artifact_uuid       TEXT    not null
        constraint unique_artifact_uuid
            unique,
    constraint unique_artifact_image_id_and_version
        unique (artifact_image_id, artifact_version)
);

INSERT INTO artifacts_original (
    artifact_id,
    artifact_version,
    artifact_image_id,
    artifact_created_at,
    artifact_updated_at,
    artifact_created_by,
    artifact_updated_by,
    artifact_metadata,
    artifact_uuid
)
SELECT 
    artifact_id,
    artifact_version,
    artifact_image_id,
    artifact_created_at,
    artifact_updated_at,
    artifact_created_by,
    artifact_updated_by,
    artifact_metadata,
    artifact_uuid
FROM artifacts;

DROP TABLE artifacts;

ALTER TABLE artifacts_original RENAME TO artifacts;

CREATE INDEX idx_artifacts_artifact_image_id
    ON artifacts (artifact_image_id);
