
-- Recreate registries table without UUID column
CREATE TABLE registries_old
(
    registry_id               INTEGER PRIMARY KEY autoincrement,
    registry_name             TEXT    NOT NULL,
    registry_root_parent_id   INTEGER NOT NULL,
    registry_parent_id        INTEGER NOT NULL,
    registry_description      TEXT,
    registry_type             TEXT    NOT NULL,
    registry_package_type     TEXT    NOT NULL,
    registry_upstream_proxies TEXT,
    registry_allowed_pattern  TEXT,
    registry_blocked_pattern  TEXT,
    registry_labels           TEXT,
    registry_created_at       BIGINT  NOT NULL,
    registry_updated_at       BIGINT  NOT NULL,
    registry_created_by       INTEGER NOT NULL,
    registry_updated_by       INTEGER NOT NULL,

    CONSTRAINT unique_registries
        UNIQUE (registry_root_parent_id, registry_name),
    CONSTRAINT unique_registry_parent_name
        UNIQUE (registry_parent_id, registry_name),
    CONSTRAINT registry_name_len_check
        CHECK (length(registry_name) <= 255)
);

INSERT INTO registries_old (
    registry_id, registry_name, registry_root_parent_id, registry_parent_id,
    registry_description, registry_type, registry_package_type, registry_upstream_proxies,
    registry_allowed_pattern, registry_blocked_pattern, registry_labels,
    registry_created_at, registry_updated_at, registry_created_by, registry_updated_by
)
SELECT
    registry_id, registry_name, registry_root_parent_id, registry_parent_id,
    registry_description, registry_type, registry_package_type, registry_upstream_proxies,
    registry_allowed_pattern, registry_blocked_pattern, registry_labels, registry_created_at, registry_updated_at,
    registry_created_by, registry_updated_by
FROM registries;

DROP TABLE registries;
ALTER TABLE registries_old RENAME TO registries;


-- Recreate images table without UUID column
CREATE TABLE images_old
(
    image_id          INTEGER PRIMARY KEY autoincrement,
    image_name        TEXT    NOT NULL,
    image_registry_id INTEGER NOT NULL
        CONSTRAINT fk_registries_registry_id
            REFERENCES registries
            ON DELETE CASCADE,
    image_labels      TEXT,
    image_enabled     BOOLEAN DEFAULT FALSE,
    image_created_at  INTEGER NOT NULL,
    image_updated_at  INTEGER NOT NULL,
    image_created_by  INTEGER NOT NULL,
    image_updated_by  INTEGER NOT NULL,

    CONSTRAINT unique_image_registry_id_and_name
        UNIQUE (image_registry_id, image_name),
    CONSTRAINT check_image_name_length
        CHECK ((LENGTH(image_name) <= 255))
);

INSERT INTO images_old (
    image_id, image_name, image_registry_id, image_labels, image_enabled, image_created_at,
    image_updated_at, image_created_by, image_updated_by
)
SELECT
    image_id, image_name, image_registry_id, image_labels, image_enabled, image_created_at,
    image_updated_at, image_created_by, image_updated_by
FROM images;

DROP TABLE images;
ALTER TABLE images_old RENAME TO images;


-- Recreate artifacts table without UUID column
CREATE TABLE artifacts_old
(
    artifact_id         INTEGER PRIMARY KEY autoincrement,
    artifact_version    TEXT    NOT NULL,
    artifact_image_id   INTEGER NOT NULL
        CONSTRAINT fk_images_image_id
            REFERENCES images
            ON DELETE CASCADE,
    artifact_created_at INTEGER NOT NULL,
    artifact_updated_at INTEGER NOT NULL,
    artifact_created_by INTEGER NOT NULL,
    artifact_updated_by INTEGER NOT NULL,
    artifact_metadata   TEXT,

    CONSTRAINT unique_artifact_image_id_and_version
        UNIQUE (artifact_image_id, artifact_version)
);

INSERT INTO artifacts_old (
    artifact_id, artifact_version, artifact_image_id, artifact_created_at,
    artifact_updated_at, artifact_created_by, artifact_updated_by, artifact_metadata
)
SELECT
    artifact_id, artifact_version, artifact_image_id, artifact_created_at,
    artifact_updated_at, artifact_created_by, artifact_updated_by, artifact_metadata
FROM artifacts;

DROP TABLE artifacts;
ALTER TABLE artifacts_old RENAME TO artifacts;
