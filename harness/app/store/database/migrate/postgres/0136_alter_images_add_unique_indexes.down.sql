DROP INDEX IF EXISTS unique_images_with_type;
DROP INDEX IF EXISTS unique_images_without_type;

ALTER TABLE images ALTER COLUMN image_type TYPE TEXT USING image_type::image_type_enum::text;
DROP TYPE IF EXISTS image_type_enum;

ALTER TABLE images ADD CONSTRAINT unique_images UNIQUE (image_registry_id, image_name, image_type);
