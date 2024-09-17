CREATE TABLE artifacts_temp
(
    artifact_id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    artifact_name                TEXT NOT NULL,
    artifact_registry_id         INTEGER NOT NULL
        CONSTRAINT fk_registries_registry_id
            REFERENCES registries(registry_id)
            ON DELETE CASCADE,
    artifact_labels              TEXT,
    artifact_enabled             BOOLEAN DEFAULT FALSE,
    artifact_created_at          INTEGER,
    artifact_updated_at          INTEGER,
    artifact_created_by          INTEGER,
    artifact_updated_by          INTEGER,
    CONSTRAINT unique_artifact_registry_id_and_name UNIQUE (artifact_registry_id, artifact_name),
    CONSTRAINT check_artifact_name_length CHECK ((LENGTH(artifact_name) <= 255))
);

INSERT INTO artifacts_temp (artifact_name, artifact_registry_id, artifact_labels)
SELECT i.image_name AS artifact_name,
       i.image_registry_id AS artifact_registry_id,
       i.images_labels AS artifact_labels
FROM artifacts a
         JOIN images i ON a.artifact_image_id = i.image_id;

DROP TABLE artifacts;

ALTER TABLE artifacts_temp
    RENAME TO artifacts;

CREATE INDEX index_artifact_on_registry_id ON artifacts (artifact_registry_id);

DROP TABLE images;
