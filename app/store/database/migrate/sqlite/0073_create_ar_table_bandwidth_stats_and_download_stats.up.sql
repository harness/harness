CREATE TABLE download_stats
(
    download_stat_id                               INTEGER PRIMARY KEY AUTOINCREMENT,
    download_stat_artifact_id                      INTEGER NOT NULL
        CONSTRAINT fk_artifacts_artifact_id
            REFERENCES artifacts(artifact_id) ,
    download_stat_timestamp                        INTEGER NOT NULL,
    download_stat_created_at                       INTEGER NOT NULL,
    download_stat_updated_at                       INTEGER NOT NULL,
    download_stat_created_by                       INTEGER NOT NULL,
    download_stat_updated_by                       INTEGER NOT NULL
);

CREATE TABLE bandwidth_stats
(
    bandwidth_stat_id                               INTEGER PRIMARY KEY AUTOINCREMENT,
    bandwidth_stat_image_id                         INTEGER NOT NULL
        CONSTRAINT fk_images_image_id
            REFERENCES images(image_id) ,
    bandwidth_stat_timestamp                        INTEGER NOT NULL,
    bandwidth_stat_bytes                            INTEGER,
    bandwidth_stat_type                             TEXT NOT NULL,
    bandwidth_stat_created_at                       INTEGER NOT NULL,
    bandwidth_stat_updated_at                       INTEGER NOT NULL,
    bandwidth_stat_created_by                       INTEGER NOT NULL,
    bandwidth_stat_updated_by                       INTEGER NOT NULL
);