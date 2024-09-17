CREATE TABLE images
(
    image_id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    image_name                TEXT NOT NULL,
    image_registry_id         INTEGER NOT NULL
        CONSTRAINT fk_registries_registry_id
            REFERENCES registries(registry_id),
    image_labels              text,
    image_enabled             BOOLEAN DEFAULT FALSE,
    image_created_at          INTEGER NOT NULL,
    image_updated_at          INTEGER NOT NULL,
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
    artifact_id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    artifact_version             TEXT NOT NULL,
    artifact_image_id            INTEGER NOT NULL
        CONSTRAINT fk_images_image_id
            REFERENCES images(image_id),
    artifact_created_at          INTEGER NOT NULL,
    artifact_updated_at          INTEGER NOT NULL,
    artifact_created_by          INTEGER NOT NULL,
    artifact_updated_by          INTEGER NOT NULL,
    CONSTRAINT unique_artifact_image_id_and_version UNIQUE (artifact_image_id, artifact_version)
);

INSERT INTO artifacts_temp (artifact_image_id, artifact_version, artifact_created_at, artifact_updated_at,
                            artifact_created_by, artifact_updated_by)
SELECT i.image_id AS artifact_image_id,
       m.manifest_digest AS artifact_version,
       m.manifest_created_at AS artifact_created_at,
       m.manifest_updated_at AS artifact_updated_at,
       m.manifest_created_by AS artifact_created_by,
       m.manifest_updated_by AS artifact_updated_by
FROM artifacts a
    JOIN images i ON a.artifact_name = i.image_name AND a.artifact_registry_id = i.image_registry_id
    JOIN manifests m ON a.artifact_name = m.manifest_image_name AND a.artifact_registry_id = m.manifest_registry_id;


DROP INDEX index_artifact_on_registry_id;
DROP TABLE artifacts;

ALTER TABLE artifacts_temp
    RENAME TO artifacts;