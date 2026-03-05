-- Drop indexes first
DROP INDEX IF EXISTS idx_registries_root_parent_id_deleted_at;
DROP INDEX IF EXISTS idx_images_registry_id_deleted_at;
DROP INDEX IF EXISTS idx_artifacts_image_id_deleted_at;

-- Drop columns from registries table
ALTER TABLE registries DROP COLUMN registry_deleted_at;
ALTER TABLE registries DROP COLUMN registry_deleted_by;

-- Drop columns from images table
ALTER TABLE images DROP COLUMN image_deleted_at;
ALTER TABLE images DROP COLUMN image_deleted_by;

-- Drop columns from artifacts table
ALTER TABLE artifacts DROP COLUMN artifact_deleted_at;
ALTER TABLE artifacts DROP COLUMN artifact_deleted_by;
