-- For registries table
ALTER TABLE registries ADD COLUMN registry_deleted_at INTEGER;
ALTER TABLE registries ADD COLUMN registry_deleted_by INTEGER;
-- Index for cleanup jobs to find soft-deleted registries by account
CREATE INDEX IF NOT EXISTS idx_registries_root_parent_id_deleted_at 
ON registries (registry_root_parent_id, registry_deleted_at) 
WHERE registry_deleted_at IS NOT NULL;

-- For images table
ALTER TABLE images ADD COLUMN image_deleted_at INTEGER;
ALTER TABLE images ADD COLUMN image_deleted_by INTEGER;
-- Index for cleanup jobs to find soft-deleted images
CREATE INDEX IF NOT EXISTS idx_images_registry_id_deleted_at 
ON images (image_registry_id, image_deleted_at) 
WHERE image_deleted_at IS NOT NULL;

-- For artifacts table
ALTER TABLE artifacts ADD COLUMN artifact_deleted_at INTEGER;
ALTER TABLE artifacts ADD COLUMN artifact_deleted_by INTEGER;
-- Index for cleanup jobs to find soft-deleted artifacts
CREATE INDEX IF NOT EXISTS idx_artifacts_image_id_deleted_at 
ON artifacts (artifact_image_id, artifact_deleted_at) 
WHERE artifact_deleted_at IS NOT NULL;
