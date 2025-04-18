CREATE TABLE download_stats_new
(
    download_stat_id                               INTEGER PRIMARY KEY AUTOINCREMENT,
    download_stat_artifact_id                      INTEGER NOT NULL 
      CONSTRAINT fk_artifacts_artifact_id  
        REFERENCES artifacts (artifact_id),
    download_stat_timestamp                        INTEGER NOT NULL,
    download_stat_created_at                       INTEGER NOT NULL,
    download_stat_updated_at                       INTEGER NOT NULL,
    download_stat_created_by                       INTEGER NOT NULL,
    download_stat_updated_by                       INTEGER NOT NULL
);

DROP INDEX IF EXISTS download_stat_artifact_id;

CREATE INDEX download_stat_artifact_id ON download_stats_new(download_stat_artifact_id);

INSERT INTO download_stats_new (
    download_stat_id, 
    download_stat_artifact_id, 
    download_stat_timestamp,
    download_stat_created_at, 
    download_stat_updated_at,
    download_stat_created_by,
    download_stat_updated_by
)
SELECT
    download_stat_id, 
    download_stat_artifact_id, 
    download_stat_timestamp,
    download_stat_created_at, 
    download_stat_updated_at,
    download_stat_created_by,
    download_stat_updated_by
FROM download_stats;

DROP TABLE download_stats;

ALTER TABLE download_stats_new RENAME TO download_stats;
