ALTER TABLE images ADD COLUMN image_node_id UUID;

ALTER TABLE images ADD CONSTRAINT fk_images_node_id
    FOREIGN KEY (image_node_id) REFERENCES nodes (node_id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_images_node_id ON images (image_node_id);