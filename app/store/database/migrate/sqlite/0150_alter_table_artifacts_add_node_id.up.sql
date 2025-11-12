CREATE TABLE artifacts_new (
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
    artifact_node_id    TEXT
        constraint fk_artifacts_node_id
            references nodes (node_id)
            on delete set null,
    constraint unique_artifact_image_id_and_version
        unique (artifact_image_id, artifact_version)
);

INSERT INTO artifacts_new (
    artifact_id,
    artifact_version,
    artifact_image_id,
    artifact_created_at,
    artifact_updated_at,
    artifact_created_by,
    artifact_updated_by,
    artifact_metadata,
    artifact_uuid,
    artifact_node_id
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
    artifact_uuid,
    NULL AS artifact_node_id
FROM artifacts;

DROP TABLE artifacts;

ALTER TABLE artifacts_new RENAME TO artifacts;

CREATE INDEX idx_artifacts_artifact_image_id
    ON artifacts (artifact_image_id);

CREATE INDEX IF NOT EXISTS idx_artifacts_node_id ON artifacts (artifact_node_id);
