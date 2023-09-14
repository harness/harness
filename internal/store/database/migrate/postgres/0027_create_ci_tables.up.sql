CREATE TABLE pipelines (
    pipeline_id SERIAL PRIMARY KEY,
    pipeline_description TEXT NOT NULL,
    pipeline_uid TEXT NOT NULL,
    pipeline_seq INTEGER NOT NULL DEFAULT 0,
    pipeline_disabled BOOLEAN NOT NULL,
    pipeline_repo_id INTEGER NOT NULL,
    pipeline_default_branch TEXT NOT NULL,
    pipeline_created_by INTEGER NOT NULL,
    pipeline_config_path TEXT NOT NULL,
    pipeline_created BIGINT NOT NULL,
    pipeline_updated BIGINT NOT NULL,
    pipeline_version INTEGER NOT NULL,
    UNIQUE (pipeline_repo_id, pipeline_uid),
    CONSTRAINT fk_pipelines_repo_id FOREIGN KEY (pipeline_repo_id)
        REFERENCES repositories (repo_id) ON DELETE CASCADE,
    CONSTRAINT fk_pipelines_created_by FOREIGN KEY (pipeline_created_by)
        REFERENCES principals (principal_id) ON DELETE NO ACTION
);

CREATE TABLE executions (
    execution_id SERIAL PRIMARY KEY,
    execution_pipeline_id INTEGER NOT NULL,
    execution_repo_id INTEGER NOT NULL,
    execution_created_by INTEGER NOT NULL,
    execution_trigger TEXT NOT NULL,
    execution_number INTEGER NOT NULL,
    execution_parent INTEGER NOT NULL,
    execution_status TEXT NOT NULL,
    execution_error TEXT NOT NULL,
    execution_event TEXT NOT NULL,
    execution_action TEXT NOT NULL,
    execution_link TEXT NOT NULL,
    execution_timestamp INTEGER NOT NULL,
    execution_title TEXT NOT NULL,
    execution_message TEXT NOT NULL,
    execution_before TEXT NOT NULL,
    execution_after TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    execution_source_repo TEXT NOT NULL,
    execution_source TEXT NOT NULL,
    execution_target TEXT NOT NULL,
    execution_author TEXT NOT NULL,
    execution_author_name TEXT NOT NULL,
    execution_author_email TEXT NOT NULL,
    execution_author_avatar TEXT NOT NULL,
    execution_sender TEXT NOT NULL,
    execution_params TEXT NOT NULL,
    execution_cron TEXT NOT NULL,
    execution_deploy TEXT NOT NULL,
    execution_deploy_id INTEGER NOT NULL,
    execution_debug BOOLEAN NOT NULL DEFAULT false,
    execution_started BIGINT NOT NULL,
    execution_finished BIGINT NOT NULL,
    execution_created BIGINT NOT NULL,
    execution_updated BIGINT NOT NULL,
    execution_version INTEGER NOT NULL,
    UNIQUE (execution_pipeline_id, execution_number),
    CONSTRAINT fk_executions_pipeline_id FOREIGN KEY (execution_pipeline_id)
        REFERENCES pipelines (pipeline_id) ON DELETE CASCADE,
    CONSTRAINT fk_executions_repo_id FOREIGN KEY (execution_repo_id)
        REFERENCES repositories (repo_id) ON DELETE CASCADE,
    CONSTRAINT fk_executions_created_by FOREIGN KEY (execution_created_by)
        REFERENCES principals (principal_id) ON DELETE NO ACTION
);

CREATE TABLE secrets (
    secret_id SERIAL PRIMARY KEY,
    secret_uid TEXT NOT NULL,
    secret_space_id INTEGER NOT NULL,
    secret_description TEXT NOT NULL,
    secret_data BYTEA NOT NULL,
    secret_created_by INTEGER NOT NULL,
    secret_created BIGINT NOT NULL,
    secret_updated BIGINT NOT NULL,
    secret_version INTEGER NOT NULL,
    UNIQUE (secret_space_id, secret_uid),
    CONSTRAINT fk_secrets_space_id FOREIGN KEY (secret_space_id)
        REFERENCES spaces (space_id) ON DELETE CASCADE,
    CONSTRAINT fk_secrets_created_by FOREIGN KEY (secret_created_by)
        REFERENCES principals (principal_id) ON DELETE NO ACTION
);

CREATE TABLE stages (
    stage_id SERIAL PRIMARY KEY,
    stage_execution_id INTEGER NOT NULL,
    stage_repo_id INTEGER NOT NULL,
    stage_number INTEGER NOT NULL,
    stage_kind TEXT NOT NULL,
    stage_type TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    stage_status TEXT NOT NULL,
    stage_error TEXT NOT NULL,
    stage_parent_group_id INTEGER NOT NULL,
    stage_errignore BOOLEAN NOT NULL,
    stage_exit_code INTEGER NOT NULL,
    stage_limit INTEGER NOT NULL,
    stage_os TEXT NOT NULL,
    stage_arch TEXT NOT NULL,
    stage_variant TEXT NOT NULL,
    stage_kernel TEXT NOT NULL,
    stage_machine TEXT NOT NULL,
    stage_started BIGINT NOT NULL,
    stage_stopped BIGINT NOT NULL,
    stage_created BIGINT NOT NULL,
    stage_updated BIGINT NOT NULL,
    stage_version INTEGER NOT NULL,
    stage_on_success BOOLEAN NOT NULL,
    stage_on_failure BOOLEAN NOT NULL,
    stage_depends_on TEXT NOT NULL,
    stage_labels TEXT NOT NULL,
    stage_limit_repo INTEGER NOT NULL DEFAULT 0,
    UNIQUE (stage_execution_id, stage_number),
    CONSTRAINT fk_stages_execution_id FOREIGN KEY (stage_execution_id)
        REFERENCES executions (execution_id) ON DELETE CASCADE
);

CREATE INDEX ix_stage_in_progress ON stages (stage_status)
WHERE stage_status IN ('pending', 'running');

CREATE TABLE steps (
    step_id SERIAL PRIMARY KEY,
    step_stage_id INTEGER NOT NULL,
    step_number INTEGER NOT NULL,
    step_name TEXT NOT NULL,
    step_status TEXT NOT NULL,
    step_error TEXT NOT NULL,
    step_parent_group_id INTEGER NOT NULL,
    step_errignore BOOLEAN NOT NULL,
    step_exit_code INTEGER NOT NULL,
    step_started BIGINT NOT NULL,
    step_stopped BIGINT NOT NULL,
    step_version INTEGER NOT NULL,
    step_depends_on TEXT NOT NULL,
    step_image TEXT NOT NULL,
    step_detached BOOLEAN NOT NULL,
    step_schema TEXT NOT NULL,
    UNIQUE (step_stage_id, step_number),
    CONSTRAINT fk_steps_stage_id FOREIGN KEY (step_stage_id)
        REFERENCES stages (stage_id) ON DELETE CASCADE
);

CREATE TABLE logs (
    log_id SERIAL PRIMARY KEY,
    log_data BYTEA NOT NULL,
    CONSTRAINT fk_logs_id FOREIGN KEY (log_id)
        REFERENCES steps (step_id) ON DELETE CASCADE
);

CREATE TABLE connectors (
    connector_id SERIAL PRIMARY KEY,
    connector_uid TEXT NOT NULL,
    connector_description TEXT NOT NULL,
    connector_type TEXT NOT NULL,
    connector_space_id INTEGER NOT NULL,
    connector_data TEXT NOT NULL,
    connector_created BIGINT NOT NULL,
    connector_updated BIGINT NOT NULL,
    connector_version INTEGER NOT NULL,
    UNIQUE (connector_space_id, connector_uid),
    CONSTRAINT fk_connectors_space_id FOREIGN KEY (connector_space_id)
        REFERENCES spaces (space_id) ON DELETE CASCADE
);

CREATE TABLE templates (
    template_id SERIAL PRIMARY KEY,
    template_uid TEXT NOT NULL,
    template_description TEXT NOT NULL,
    template_space_id INTEGER NOT NULL,
    template_data TEXT NOT NULL,
    template_created BIGINT NOT NULL,
    template_updated BIGINT NOT NULL,
    template_version INTEGER NOT NULL,
    UNIQUE (template_space_id, template_uid),
    CONSTRAINT fk_templates_space_id FOREIGN KEY (template_space_id)
        REFERENCES spaces (space_id) ON DELETE CASCADE
);

CREATE TABLE triggers (
    trigger_id SERIAL PRIMARY KEY,
    trigger_uid TEXT NOT NULL,
    trigger_pipeline_id INTEGER NOT NULL,
    trigger_type TEXT NOT NULL,
    trigger_repo_id INTEGER NOT NULL,
    trigger_secret TEXT NOT NULL,
    trigger_description TEXT NOT NULL,
    trigger_disabled BOOLEAN NOT NULL,
    trigger_created_by INTEGER NOT NULL,
    trigger_actions TEXT NOT NULL,
    trigger_created BIGINT NOT NULL,
    trigger_updated BIGINT NOT NULL,
    trigger_version INTEGER NOT NULL,
    UNIQUE (trigger_pipeline_id, trigger_uid),
    CONSTRAINT fk_triggers_pipeline_id FOREIGN KEY (trigger_pipeline_id)
        REFERENCES pipelines (pipeline_id) ON DELETE CASCADE,
    CONSTRAINT fk_triggers_repo_id FOREIGN KEY (trigger_repo_id)
        REFERENCES repositories (repo_id) ON DELETE CASCADE
);

CREATE TABLE plugins (
    plugin_uid TEXT NOT NULL,
    plugin_description TEXT NOT NULL,
    plugin_logo TEXT NOT NULL,
    plugin_spec BYTEA NOT NULL,
    UNIQUE (plugin_uid)
);