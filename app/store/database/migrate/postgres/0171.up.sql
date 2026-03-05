-- Add deleted_at and deleted_by columns to registries, images, and artifacts tables
-- Add to registries table
ALTER TABLE registries
    ADD COLUMN IF NOT EXISTS registry_deleted_at BIGINT,
    ADD COLUMN IF NOT EXISTS registry_deleted_by INTEGER;

-- Index for cleanup jobs to find soft-deleted registries by account
-- Supports: WHERE registry_root_parent_id = ? AND registry_deleted_at IS NOT NULL AND registry_deleted_at <= ?
CREATE INDEX IF NOT EXISTS idx_registries_root_parent_id_deleted_at
ON registries (registry_root_parent_id, registry_deleted_at) 
WHERE registry_deleted_at IS NOT NULL;

-- Add to images table
ALTER TABLE images
    ADD COLUMN IF NOT EXISTS image_deleted_at BIGINT,
    ADD COLUMN IF NOT EXISTS image_deleted_by INTEGER;

-- Index for cleanup jobs to find soft-deleted images
-- Supports: JOIN registries ON image_registry_id WHERE image_deleted_at IS NOT NULL AND image_deleted_at <= ?
CREATE INDEX IF NOT EXISTS idx_images_registry_id_deleted_at
ON images (image_registry_id, image_deleted_at) 
WHERE image_deleted_at IS NOT NULL;

-- Add to artifacts table
ALTER TABLE artifacts
    ADD COLUMN IF NOT EXISTS artifact_deleted_at BIGINT,
    ADD COLUMN IF NOT EXISTS artifact_deleted_by INTEGER;

-- Index for cleanup jobs to find soft-deleted artifacts
-- Supports: JOIN images ON artifact_image_id WHERE artifact_deleted_at IS NOT NULL AND artifact_deleted_at <= ?
CREATE INDEX IF NOT EXISTS idx_artifacts_image_id_deleted_at
ON artifacts (artifact_image_id, artifact_deleted_at) 
WHERE artifact_deleted_at IS NOT NULL;
