ALTER TABLE images DROP COLUMN image_type;
ALTER TABLE images DROP CONSTRAINT IF EXISTS unique_images;
ALTER TABLE images ADD CONSTRAINT unique_image_registry_id_and_name UNIQUE (image_registry_id, image_name);