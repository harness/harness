-- Add UUID columns to tables
ALTER TABLE registries ADD COLUMN registry_uuid TEXT;
ALTER TABLE images ADD COLUMN image_uuid TEXT;
ALTER TABLE artifacts ADD COLUMN artifact_uuid TEXT;

-- Generate UUIDs for existing entries
UPDATE registries
SET registry_uuid = lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(6)))
WHERE registry_uuid IS NULL;

UPDATE images
SET image_uuid = lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(6)))
WHERE image_uuid IS NULL;

UPDATE artifacts
SET artifact_uuid = lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(6)))
WHERE artifact_uuid IS NULL;

-- Recreate registries table with NOT NULL and UNIQUE constraints on UUID
CREATE TABLE registries_new
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
    registry_uuid TEXT NOT NULL,

    CONSTRAINT unique_registries
        UNIQUE (registry_root_parent_id, registry_name),
    CONSTRAINT unique_registry_parent_name
        UNIQUE (registry_parent_id, registry_name),
    CONSTRAINT registry_name_len_check
        CHECK (length(registry_name) <= 255),
    CONSTRAINT unique_registry_uuid UNIQUE (registry_uuid)
);

INSERT INTO registries_new 
SELECT * FROM registries;

DROP TABLE registries;
ALTER TABLE registries_new RENAME TO registries;

-- Recreate images table with NOT NULL and UNIQUE constraints on UUID
CREATE TABLE images_new
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
    image_uuid TEXT NOT NULL,

    CONSTRAINT unique_image_registry_id_and_name
        UNIQUE (image_registry_id, image_name),
    CONSTRAINT check_image_name_length
        CHECK ((LENGTH(image_name) <= 255)),
    CONSTRAINT unique_image_uuid UNIQUE (image_uuid)
);

INSERT INTO images_new 
SELECT * FROM images;

DROP TABLE images;
ALTER TABLE images_new RENAME TO images;

-- Recreate artifacts table with NOT NULL and UNIQUE constraints on UUID
CREATE TABLE artifacts_new
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
    artifact_uuid TEXT NOT NULL,

    CONSTRAINT unique_artifact_image_id_and_version
        UNIQUE (artifact_image_id, artifact_version),
    CONSTRAINT unique_artifact_uuid UNIQUE (artifact_uuid)
);

INSERT INTO artifacts_new 
SELECT * FROM artifacts;

DROP TABLE artifacts;
ALTER TABLE artifacts_new RENAME TO artifacts;
