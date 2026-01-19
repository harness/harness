ALTER TABLE usage_metrics
    ALTER COLUMN usage_metric_bandwidth_out DROP DEFAULT,
    ALTER COLUMN usage_metric_storage_total DROP DEFAULT,
    ADD COLUMN usage_metric_version BIGINT NOT NULL DEFAULT 0;
