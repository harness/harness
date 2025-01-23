CREATE TABLE registry_webhooks
(
    registry_webhook_id SERIAL PRIMARY KEY,
    registry_webhook_version INTEGER NOT NULL DEFAULT 0,
    registry_webhook_created_by INTEGER NOT NULL,
    registry_webhook_created BIGINT NOT NULL,
    registry_webhook_updated BIGINT NOT NULL,
    registry_webhook_space_id INTEGER,
    registry_webhook_registry_id INTEGER,
    registry_webhook_name TEXT NOT NULL,
    registry_webhook_description TEXT NOT NULL,
    registry_webhook_url TEXT NOT NULL,
    registry_webhook_secret_identifier TEXT,
    registry_webhook_secret_space_id INTEGER,
    registry_webhook_enabled BOOLEAN NOT NULL,
    registry_webhook_insecure BOOLEAN NOT NULL,
    registry_webhook_triggers TEXT NOT NULL,
    registry_webhook_latest_execution_result TEXT,
    registry_webhook_scope INTEGER DEFAULT 0,
    registry_webhook_internal BOOLEAN NOT NULL,
    registry_webhook_identifier TEXT NOT NULL,
    registry_webhook_extra_headers TEXT,
    CONSTRAINT fk_registry_webhook_created_by FOREIGN KEY (registry_webhook_created_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION,
    CONSTRAINT fk_registry_webhook_registry_id FOREIGN KEY (registry_webhook_registry_id)
    REFERENCES registries (registry_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
    );

CREATE UNIQUE INDEX registry_webhooks_registry_id_identifier
    ON registry_webhooks(registry_webhook_registry_id, registry_webhook_identifier)
    WHERE registry_webhook_space_id IS NULL;

CREATE UNIQUE INDEX registry_webhooks_space_id_identifier
    ON registry_webhooks(registry_webhook_space_id, registry_webhook_identifier)
    WHERE registry_webhook_registry_id IS NULL;


CREATE TABLE registry_webhook_executions
(
    registry_webhook_execution_id SERIAL PRIMARY KEY,
    registry_webhook_execution_retrigger_of INTEGER,
    registry_webhook_execution_retriggerable BOOLEAN NOT NULL,
    registry_webhook_execution_webhook_id INTEGER NOT NULL,
    registry_webhook_execution_trigger_type TEXT NOT NULL,
    registry_webhook_execution_trigger_id TEXT NOT NULL,
    registry_webhook_execution_result TEXT NOT NULL,
    registry_webhook_execution_created BIGINT NOT NULL,
    registry_webhook_execution_duration BIGINT NOT NULL,
    registry_webhook_execution_error TEXT NOT NULL,
    registry_webhook_execution_request_url TEXT NOT NULL,
    registry_webhook_execution_request_headers TEXT NOT NULL,
    registry_webhook_execution_request_body TEXT NOT NULL,
    registry_webhook_execution_response_status_code INTEGER NOT NULL,
    registry_webhook_execution_response_status TEXT NOT NULL,
    registry_webhook_execution_response_headers TEXT NOT NULL,
    registry_webhook_execution_response_body TEXT NOT NULL
);