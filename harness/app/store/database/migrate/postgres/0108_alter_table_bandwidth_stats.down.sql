ALTER TABLE bandwidth_stats DROP CONSTRAINT IF EXISTS fk_images_image_id;

ALTER TABLE bandwidth_stats
ADD CONSTRAINT fk_images_image_id
FOREIGN KEY (bandwidth_stat_image_id) 
REFERENCES images(image_id);