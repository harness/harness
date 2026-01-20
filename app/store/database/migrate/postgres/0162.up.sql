ALTER TABLE usage_metrics
    ALTER COLUMN usage_metric_bandwidth_out SET DEFAULT 0,
    ALTER COLUMN usage_metric_storage_total SET DEFAULT 0,
    DROP COLUMN usage_metric_version;
