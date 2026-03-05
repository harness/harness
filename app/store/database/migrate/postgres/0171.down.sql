-- Remove from artifacts table
DROP INDEX IF EXISTS idx_artifacts_image_id_deleted_at;
ALTER TABLE artifacts 
    DROP COLUMN IF EXISTS artifact_deleted_at,
    DROP COLUMN IF EXISTS artifact_deleted_by;

-- Remove from images table
DROP INDEX IF EXISTS idx_images_registry_id_deleted_at;
ALTER TABLE images 
    DROP COLUMN IF EXISTS image_deleted_at,
    DROP COLUMN IF EXISTS image_deleted_by;

-- Remove from registries table
DROP INDEX IF EXISTS idx_registries_root_parent_id_deleted_at;
ALTER TABLE registries 
    DROP COLUMN IF EXISTS registry_deleted_at,
    DROP COLUMN IF EXISTS registry_deleted_by;
