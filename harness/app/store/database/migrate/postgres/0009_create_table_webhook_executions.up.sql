CREATE TABLE webhook_executions (
webhook_execution_id SERIAL PRIMARY KEY
,webhook_execution_retrigger_of INTEGER
,webhook_execution_retriggerable BOOLEAN NOT NULL
,webhook_execution_webhook_id INTEGER NOT NULL
,webhook_execution_trigger_type TEXT NOT NULL
,webhook_execution_trigger_id TEXT NOT NULL
,webhook_execution_result TEXT NOT NULL
,webhook_execution_created BIGINT NOT NULL
,webhook_execution_duration BIGINT NOT NULL
,webhook_execution_error TEXT NOT NULL
,webhook_execution_request_url TEXT NOT NULL
,webhook_execution_request_headers TEXT NOT NULL
,webhook_execution_request_body TEXT NOT NULL
,webhook_execution_response_status_code INTEGER NOT NULL
,webhook_execution_response_status TEXT NOT NULL
,webhook_execution_response_headers TEXT NOT NULL
,webhook_execution_response_body TEXT NOT NULL
);
