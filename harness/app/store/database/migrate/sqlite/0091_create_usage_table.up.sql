CREATE TABLE usage_metrics
(
    usage_metric_space_id  BIGINT NOT NULL,
	usage_metric_date      BIGINT NOT NULL,
	usage_metric_created   BIGINT NOT NULL,
	usage_metric_updated   BIGINT,
	usage_metric_bandwidth BIGINT NOT NULL,
	usage_metric_storage   BIGINT NOT NULL,
    usage_metric_version   BIGINT NOT NULL,

	PRIMARY KEY (usage_metric_space_id, usage_metric_date),

	CONSTRAINT fk_usagemetric_space_id FOREIGN KEY (usage_metric_space_id)
		REFERENCES spaces (space_id) MATCH SIMPLE
		ON UPDATE NO ACTION
		ON DELETE CASCADE
);