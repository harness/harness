create table images_new
(
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
    constraint check_image_name_length
        check ((LENGTH(image_name) <= 255))
);

CREATE UNIQUE INDEX unique_images_with_type ON images_new (image_registry_id, image_name, image_type) WHERE image_type IS NOT NULL;
CREATE UNIQUE INDEX unique_images_without_type ON images_new (image_registry_id, image_name) WHERE image_type IS NULL;

INSERT INTO images_new (
    image_id, image_name, image_registry_id, image_labels, image_enabled, image_created_at,
    image_updated_at, image_created_by, image_updated_by, image_uuid
)
SELECT
    image_id, image_name, image_registry_id, image_labels, image_enabled, image_created_at,
    image_updated_at, image_created_by, image_updated_by, image_uuid
FROM images;

DROP TABLE images;
ALTER TABLE images_new RENAME TO images;