ALTER TABLE repositories
    DROP COLUMN repo_lfs_size;

ALTER TABLE usage_metrics
    RENAME COLUMN usage_metric_bandwidth_out TO usage_metric_bandwidth;

ALTER TABLE usage_metrics
    DROP COLUMN usage_metric_bandwidth_in;

ALTER TABLE usage_metrics
    RENAME COLUMN usage_metric_storage_total TO usage_metric_storage;

ALTER TABLE usage_metrics
    DROP COLUMN usage_metric_lfs_storage_total;
