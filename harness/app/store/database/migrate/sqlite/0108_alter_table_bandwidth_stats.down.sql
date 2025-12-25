CREATE TABLE bandwidth_stats_new
(
    bandwidth_stat_id                               INTEGER PRIMARY KEY AUTOINCREMENT,
    bandwidth_stat_image_id                         INTEGER NOT NULL
        CONSTRAINT fk_images_image_id
            REFERENCES images(image_id),
    bandwidth_stat_timestamp                        INTEGER NOT NULL,
    bandwidth_stat_bytes                            INTEGER,
    bandwidth_stat_type                             TEXT NOT NULL,
    bandwidth_stat_created_at                       INTEGER NOT NULL,
    bandwidth_stat_updated_at                       INTEGER NOT NULL,
    bandwidth_stat_created_by                       INTEGER NOT NULL,
    bandwidth_stat_updated_by                       INTEGER NOT NULL
);

INSERT INTO bandwidth_stats_new (
    bandwidth_stat_id,
    bandwidth_stat_image_id,
    bandwidth_stat_timestamp,
    bandwidth_stat_bytes,
    bandwidth_stat_type,
    bandwidth_stat_created_at,
    bandwidth_stat_updated_at,
    bandwidth_stat_created_by,
    bandwidth_stat_updated_by
)
SELECT
    bandwidth_stat_id,
    bandwidth_stat_image_id,
    bandwidth_stat_timestamp,
    bandwidth_stat_bytes,
    bandwidth_stat_type,
    bandwidth_stat_created_at,
    bandwidth_stat_updated_at,
    bandwidth_stat_created_by,
    bandwidth_stat_updated_by
FROM bandwidth_stats;

DROP TABLE bandwidth_stats;

ALTER TABLE bandwidth_stats_new RENAME TO bandwidth_stats;
