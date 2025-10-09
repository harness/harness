ALTER TABLE images ADD COLUMN image_type TEXT;
ALTER TABLE images DROP CONSTRAINT unique_image_registry_id_and_name;
ALTER TABLE images ADD CONSTRAINT unique_images UNIQUE (image_registry_id, image_name, image_type);

