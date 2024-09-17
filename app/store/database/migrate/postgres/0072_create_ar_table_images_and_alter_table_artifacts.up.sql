CREATE TABLE images
(
    image_id                  SERIAL PRIMARY KEY,
    image_name                TEXT NOT NULL,
    image_registry_id         INTEGER NOT NULL
        CONSTRAINT fk_registries_registry_id
            references registries(registry_id),
    image_labels              TEXT,
    image_enabled             BOOLEAN DEFAULT FALSE,
    image_created_at          BIGINT NOT NULL,
    image_updated_at          BIGINT NOT NULL,
    image_created_by          INTEGER NOT NULL,
    image_updated_by          INTEGER NOT NULL,
    CONSTRAINT unique_image_registry_id_and_name UNIQUE (image_registry_id, image_name),
    CONSTRAINT check_image_name_length CHECK ((LENGTH(image_name) <= 255))
);

DROP INDEX index_artifact_on_registry_id;

ALTER TABLE artifacts DROP CONSTRAINT check_artifact_name_length;
ALTER TABLE artifacts DROP CONSTRAINT unique_artifact_registry_id_and_name;
ALTER TABLE artifacts DROP CONSTRAINT fk_registries_registry_id;

ALTER TABLE artifacts DROP COLUMN artifact_name;
ALTER TABLE artifacts DROP COLUMN artifact_registry_id;
ALTER TABLE artifacts DROP COLUMN artifact_labels;..
ALTER TABLE artifacts DROP COLUMN artifact_enabled;

ALTER TABLE artifacts ADD COLUMN artifact_version TEXT NOT NULL;
ALTER TABLE artifacts ADD COLUMN artifact_image_id INTEGER NOT NULL;


ALTER TABLE artifacts ADD CONSTRAINT fk_images_image_id FOREIGN KEY (artifact_image_id) REFERENCES images(image_id);
ALTER TABLE artifacts ADD CONSTRAINT unique_artifact_image_id_and_version UNIQUE (artifact_image_id, artifact_version);