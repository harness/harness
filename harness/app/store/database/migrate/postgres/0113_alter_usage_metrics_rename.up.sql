ALTER TABLE repositories
    ADD COLUMN repo_lfs_size BIGINT NOT NULL DEFAULT 0;

ALTER TABLE usage_metrics
    RENAME COLUMN usage_metric_bandwidth TO usage_metric_bandwidth_out;

ALTER TABLE usage_metrics
    ADD COLUMN usage_metric_bandwidth_in BIGINT NOT NULL DEFAULT 0;

ALTER TABLE usage_metrics
    RENAME COLUMN usage_metric_storage TO usage_metric_storage_total;

ALTER TABLE usage_metrics
    ADD COLUMN usage_metric_lfs_storage_total BIGINT NOT NULL DEFAULT 0;
