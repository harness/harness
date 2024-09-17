
CREATE INDEX index_artifact_on_registry_id ON artifacts (artifact_registry_id);


ALTER TABLE artifacts DROP CONSTRAINT fk_images_image_id;
ALTER TABLE artifacts DROP CONSTRAINT unique_artifact_image_id_and_version;

ALTER TABLE artifacts DROP COLUMN artifact_image_id;
ALTER TABLE artifacts DROP COLUMN artifact_version;

ALTER TABLE artifacts ADD COLUMN artifact_name TEXT NOT NULL;
ALTER TABLE artifacts ADD COLUMN artifact_registry_id INTEGER NOT NULL;
ALTER TABLE artifacts ADD COLUMN artifact_labels TEXT;
ALTER TABLE artifacts ADD COLUMN artifact_enabled BOOLEAN DEFAULT FALSE;


ALTER TABLE artifacts ADD CONSTRAINT check_artifact_name_length CHECK ((LENGTH(artifact_name) <= 255));
ALTER TABLE artifacts ADD CONSTRAINT unique_artifact_registry_id_and_name UNIQUE (artifact_registry_id, artifact_name);
ALTER TABLE artifacts ADD CONSTRAINT fk_registries_registry_id FOREIGN KEY (artifact_registry_id)
    REFERENCES registries(registry_id) ON DELETE CASCADE;

DROP TABLE images;
