DROP INDEX IF EXISTS idx_images_node_id;

ALTER TABLE images DROP CONSTRAINT IF EXISTS fk_images_node_id;

ALTER TABLE images DROP COLUMN IF EXISTS image_node_id;