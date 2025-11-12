CREATE TABLE images_new (
    image_id          INTEGER
        primary key autoincrement,
    image_name        TEXT    not null,
    image_registry_id INTEGER not null
        constraint fk_registries_registry_id
            references registries
            on delete cascade,
    image_type        TEXT,
    image_labels      TEXT,
    image_enabled     BOOLEAN default FALSE,
    image_created_at  INTEGER not null,
    image_updated_at  INTEGER not null,
    image_created_by  INTEGER not null,
    image_updated_by  INTEGER not null,
    image_uuid        TEXT    not null
        constraint unique_image_uuid
            unique,
    image_node_id     TEXT
        constraint fk_images_node_id
            references nodes (node_id)
            on delete set null,
    constraint check_image_name_length
        check ((LENGTH(image_name) <= 255))
);

INSERT INTO images_new (
    image_id,
    image_name,
    image_registry_id,
    image_type,
    image_labels,
    image_enabled,
    image_created_at,
    image_updated_at,
    image_created_by,
    image_updated_by,
    image_uuid,
    image_node_id
)
SELECT 
    image_id,
    image_name,
    image_registry_id,
    image_type,
    image_labels,
    image_enabled,
    image_created_at,
    image_updated_at,
    image_created_by,
    image_updated_by,
    image_uuid,
    NULL AS image_node_id
FROM images;

DROP TABLE images;

ALTER TABLE images_new RENAME TO images;

CREATE UNIQUE INDEX unique_images_with_type
    ON images (image_registry_id, image_name, image_type)
    WHERE image_type IS NOT NULL;

CREATE UNIQUE INDEX unique_images_without_type
    ON images (image_registry_id, image_name)
    WHERE image_type IS NULL;

CREATE INDEX IF NOT EXISTS idx_images_node_id ON images (image_node_id);
