CREATE TABLE IF NOT EXISTS executions (
    execution_id INTEGER PRIMARY KEY AUTOINCREMENT,
    execution_pipeline_id INTEGER NOT NULL,
    execution_repo_id INTEGER,
    execution_repo_type TEXT,
    execution_repo_name TEXT,
    execution_trigger TEXT,
    execution_number INTEGER NOT NULL,
    execution_parent INTEGER,
    execution_status TEXT,
    execution_error TEXT,
    execution_event TEXT,
    execution_action TEXT,
    execution_link TEXT,
    execution_timestamp INTEGER,
    execution_title TEXT,
    execution_message TEXT,
    execution_before TEXT,
    execution_after TEXT,
    execution_ref TEXT,
    execution_source_repo TEXT,
    execution_source TEXT,
    execution_target TEXT,
    execution_author TEXT,
    execution_author_name TEXT,
    execution_author_email TEXT,
    execution_author_avatar TEXT,
    execution_sender TEXT,
    execution_params TEXT,
    execution_cron TEXT,
    execution_deploy TEXT,
    execution_deploy_id INTEGER,
    execution_debug BOOLEAN NOT NULL DEFAULT 0,
    execution_started INTEGER,
    execution_finished INTEGER,
    execution_created INTEGER,
    execution_updated INTEGER,
    execution_version INTEGER,

    -- Ensure unique combination of pipeline ID and number
    UNIQUE (execution_pipeline_id, execution_number),

    -- Foreign key to pipelines table
    CONSTRAINT fk_execution_pipeline_id FOREIGN KEY (execution_pipeline_id)
        REFERENCES pipelines (pipeline_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);