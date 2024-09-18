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

INSERT INTO artifacts_temp (artifact_name, artifact_registry_id, artifact_labels, artifact_enabled,
                            artifact_created_at, artifact_updated_at, artifact_created_by, artifact_updated_by)
SELECT i.image_name AS artifact_name,
       i.image_registry_id AS artifact_registry_id,
       i.image_labels AS artifact_labels,
       i.image_enabled AS artifact_enabled,
       i.image_created_at AS artifact_created_at,
       i.image_updated_at AS artifact_updated_at,
       i.image_created_by AS artifact_created_by,
       i.image_updated_by AS artifact_updated_by

FROM artifacts a
         JOIN images i ON a.artifact_image_id = i.image_id;

ALTER TABLE artifact_stats
    DROP CONSTRAINT fk_artifacts_artifact_id;

DROP TABLE artifacts;


ALTER TABLE artifacts_temp
    RENAME TO artifacts;

ALTER TABLE artifact_stats
    ADD CONSTRAINT fk_artifacts_artifact_id FOREIGN KEY (artifact_stat_artifact_id)
        REFERENCES artifacts(artifact_id);

CREATE INDEX index_artifact_on_registry_id ON artifacts (artifact_registry_id);

DROP TABLE images;
