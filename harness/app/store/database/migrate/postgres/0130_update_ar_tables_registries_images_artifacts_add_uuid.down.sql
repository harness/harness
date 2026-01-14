ALTER TABLE artifacts
    DROP COLUMN artifact_uuid;

ALTER TABLE images
    DROP COLUMN image_uuid;

ALTER TABLE registries
    DROP COLUMN registry_uuid;