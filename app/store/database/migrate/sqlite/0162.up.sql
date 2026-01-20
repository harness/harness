CREATE TABLE usage_metrics_temp
(
    usage_metric_space_id          BIGINT            NOT NULL,
    usage_metric_date              BIGINT            NOT NULL,
    usage_metric_created           BIGINT            NOT NULL,
    usage_metric_updated           BIGINT,
    usage_metric_bandwidth_out     BIGINT  DEFAULT 0 NOT NULL,
    usage_metric_storage_total     BIGINT  DEFAULT 0 NOT NULL,
    usage_metric_pushes            INTEGER DEFAULT 0 NOT NULL,
    usage_metric_bandwidth_in      BIGINT  DEFAULT 0 NOT NULL,
    usage_metric_lfs_storage_total BIGINT  DEFAULT 0 NOT NULL,
    CONSTRAINT pk_usage_metrics
        PRIMARY KEY (usage_metric_space_id, usage_metric_date),
    CONSTRAINT fk_usagemetric_space_id
        FOREIGN KEY (usage_metric_space_id)
        REFERENCES spaces
            ON DELETE CASCADE
);

INSERT INTO usage_metrics_temp(
    usage_metric_space_id,
    usage_metric_date,
    usage_metric_created,
    usage_metric_updated,
    usage_metric_bandwidth_out,
    usage_metric_storage_total,
    usage_metric_pushes,
    usage_metric_bandwidth_in,
    usage_metric_lfs_storage_total
)
SELECT
    usage_metric_space_id,
    usage_metric_date,
    usage_metric_created,
    usage_metric_updated,
    usage_metric_bandwidth_out,
    usage_metric_storage_total,
    usage_metric_pushes,
    usage_metric_bandwidth_in,
    usage_metric_lfs_storage_total
FROM usage_metrics;

DROP TABLE usage_metrics;

ALTER TABLE usage_metrics_temp
    RENAME TO usage_metrics;