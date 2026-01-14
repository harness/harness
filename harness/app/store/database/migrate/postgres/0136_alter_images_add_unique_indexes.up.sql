ALTER TABLE images DROP CONSTRAINT IF EXISTS unique_images;

CREATE TYPE image_type_enum AS ENUM ('model', 'dataset');
ALTER TABLE images ALTER COLUMN image_type TYPE image_type_enum USING image_type::text::image_type_enum;

CREATE UNIQUE INDEX unique_images_with_type ON images (image_registry_id, image_name, image_type) WHERE image_type IS NOT NULL;
CREATE UNIQUE INDEX unique_images_without_type ON images (image_registry_id, image_name) WHERE image_type IS NULL;


