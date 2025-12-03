CREATE TABLE pipelines (
    pipeline_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,pipeline_description TEXT NOT NULL
    ,pipeline_uid TEXT NOT NULL
    ,pipeline_seq INTEGER NOT NULL DEFAULT 0
    ,pipeline_disabled BOOLEAN NOT NULL
    ,pipeline_repo_id INTEGER NOT NULL
    ,pipeline_default_branch TEXT NOT NULL
    ,pipeline_created_by INTEGER NOT NULL
    ,pipeline_config_path TEXT NOT NULL
    ,pipeline_created INTEGER NOT NULL
    ,pipeline_updated INTEGER NOT NULL
    ,pipeline_version INTEGER NOT NULL

    -- Ensure unique combination of UID and repo ID
    ,UNIQUE (pipeline_repo_id, pipeline_uid)

    -- Foreign key to repositories table
    ,CONSTRAINT fk_pipelines_repo_id FOREIGN KEY (pipeline_repo_id)
        REFERENCES repositories (repo_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE

    -- Foreign key to principals table
    ,CONSTRAINT fk_pipelines_created_by FOREIGN KEY (pipeline_created_by)
        REFERENCES principals (principal_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

CREATE TABLE executions (
    execution_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,execution_pipeline_id INTEGER NOT NULL
    ,execution_repo_id INTEGER NOT NULL
    ,execution_created_by INTEGER NOT NULL
    ,execution_trigger TEXT NOT NULL
    ,execution_number INTEGER NOT NULL
    ,execution_parent INTEGER NOT NULL
    ,execution_status TEXT NOT NULL
    ,execution_error TEXT NOT NULL
    ,execution_event TEXT NOT NULL
    ,execution_action TEXT NOT NULL
    ,execution_link TEXT NOT NULL
    ,execution_timestamp INTEGER NOT NULL
    ,execution_title TEXT NOT NULL
    ,execution_message TEXT NOT NULL
    ,execution_before TEXT NOT NULL
    ,execution_after TEXT NOT NULL
    ,execution_ref TEXT NOT NULL
    ,execution_source_repo TEXT NOT NULL
    ,execution_source TEXT NOT NULL
    ,execution_target TEXT NOT NULL
    ,execution_author TEXT NOT NULL
    ,execution_author_name TEXT NOT NULL
    ,execution_author_email TEXT NOT NULL
    ,execution_author_avatar TEXT NOT NULL
    ,execution_sender TEXT NOT NULL
    ,execution_params TEXT NOT NULL
    ,execution_cron TEXT NOT NULL
    ,execution_deploy TEXT NOT NULL
    ,execution_deploy_id INTEGER NOT NULL
    ,execution_debug BOOLEAN NOT NULL DEFAULT 0
    ,execution_started INTEGER NOT NULL
    ,execution_finished INTEGER NOT NULL
    ,execution_created INTEGER NOT NULL
    ,execution_updated INTEGER NOT NULL
    ,execution_version INTEGER NOT NULL

    -- Ensure unique combination of pipeline ID and number
    ,UNIQUE (execution_pipeline_id, execution_number)

    -- Foreign key to pipelines table
    ,CONSTRAINT fk_executions_pipeline_id FOREIGN KEY (execution_pipeline_id)
        REFERENCES pipelines (pipeline_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE

    -- Foreign key to repositories table
    ,CONSTRAINT fk_executions_repo_id FOREIGN KEY (execution_repo_id)
        REFERENCES repositories (repo_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE

    -- Foreign key to principals table
    ,CONSTRAINT fk_executions_created_by FOREIGN KEY (execution_created_by)
        REFERENCES principals (principal_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

CREATE TABLE secrets (
    secret_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,secret_uid TEXT NOT NULL
    ,secret_space_id INTEGER NOT NULL
    ,secret_description TEXT NOT NULL
    ,secret_data BLOB NOT NULL
    ,secret_created INTEGER NOT NULL
    ,secret_updated INTEGER NOT NULL
    ,secret_version INTEGER NOT NULL
    ,secret_created_by INTEGER NOT NULL

    -- Ensure unique combination of space ID and UID
    ,UNIQUE (secret_space_id, secret_uid)

    -- Foreign key to spaces table
    ,CONSTRAINT fk_secrets_space_id FOREIGN KEY (secret_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE

    -- Foreign key to principals table
    ,CONSTRAINT fk_secrets_created_by FOREIGN KEY (secret_created_by)
        REFERENCES principals (principal_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

CREATE TABLE stages (
    stage_id          INTEGER PRIMARY KEY AUTOINCREMENT
    ,stage_execution_id    INTEGER NOT NULL
    ,stage_repo_id INTEGER NOT NULL
    ,stage_number      INTEGER NOT NULL
    ,stage_kind        TEXT NOT NULL
    ,stage_type        TEXT NOT NULL
    ,stage_name        TEXT NOT NULL
    ,stage_status      TEXT NOT NULL
    ,stage_error       TEXT NOT NULL
    ,stage_parent_group_id INTEGER NOT NULL
    ,stage_errignore   BOOLEAN NOT NULL
    ,stage_exit_code   INTEGER NOT NULL
    ,stage_limit       INTEGER NOT NULL
    ,stage_os          TEXT NOT NULL
    ,stage_arch        TEXT NOT NULL
    ,stage_variant     TEXT NOT NULL
    ,stage_kernel      TEXT NOT NULL
    ,stage_machine     TEXT NOT NULL
    ,stage_started     INTEGER NOT NULL
    ,stage_stopped     INTEGER NOT NULL
    ,stage_created     INTEGER NOT NULL
    ,stage_updated     INTEGER NOT NULL
    ,stage_version     INTEGER NOT NULL
    ,stage_on_success  BOOLEAN NOT NULL
    ,stage_on_failure  BOOLEAN NOT NULL
    ,stage_depends_on  TEXT NOT NULL
    ,stage_labels      TEXT NOT NULL
    ,stage_limit_repo INTEGER NOT NULL DEFAULT 0

    -- Ensure unique combination of stage execution ID and stage number
    ,UNIQUE(stage_execution_id, stage_number)

    -- Foreign key to executions table
    ,CONSTRAINT fk_stages_execution_id FOREIGN KEY (stage_execution_id)
        REFERENCES executions (execution_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

-- name: create-index-stages-status

CREATE INDEX ix_stage_in_progress ON stages (stage_status)
WHERE stage_status IN ('pending', 'running');


CREATE TABLE steps (
    step_id          INTEGER PRIMARY KEY AUTOINCREMENT
    ,step_stage_id    INTEGER NOT NULL
    ,step_number      INTEGER NOT NULL
    ,step_name        VARCHAR(100) NOT NULL
    ,step_status      VARCHAR(50) NOT NULL
    ,step_error       VARCHAR(500) NOT NULL
    ,step_parent_group_id INTEGER NOT NULL
    ,step_errignore   BOOLEAN NOT NULL
    ,step_exit_code   INTEGER NOT NULL
    ,step_started     INTEGER NOT NULL
    ,step_stopped     INTEGER NOT NULL
    ,step_version     INTEGER NOT NULL
    ,step_depends_on  TEXT NOT NULL
    ,step_image       TEXT NOT NULL
    ,step_detached    BOOLEAN NOT NULL
    ,step_schema      TEXT NOT NULL

    -- Ensure unique comination of stage ID and step number
    ,UNIQUE(step_stage_id, step_number)

    -- Foreign key to stages table
    ,CONSTRAINT fk_steps_stage_id FOREIGN KEY (step_stage_id)
        REFERENCES stages (stage_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE 
);


CREATE TABLE logs (
    log_id INTEGER PRIMARY KEY
    ,log_data BLOB NOT NULL

    -- Foreign key to steps table
    ,CONSTRAINT fk_logs_id FOREIGN KEY (log_id)
        REFERENCES steps (step_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE connectors (
    connector_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,connector_uid TEXT NOT NULL
    ,connector_description TEXT NOT NULL
    ,connector_type TEXT NOT NULL
    ,connector_space_id INTEGER NOT NULL
    ,connector_data TEXT NOT NULL
    ,connector_created INTEGER NOT NULL
    ,connector_updated INTEGER NOT NULL
    ,connector_version INTEGER NOT NULL

    -- Ensure unique combination of space ID and UID
    ,UNIQUE (connector_space_id, connector_uid)

    -- Foreign key to spaces table
    ,CONSTRAINT fk_connectors_space_id FOREIGN KEY (connector_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE templates (
    template_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,template_uid TEXT NOT NULL
    ,template_description TEXT NOT NULL
    ,template_space_id INTEGER NOT NULL
    ,template_data TEXT NOT NULL
    ,template_created INTEGER NOT NULL
    ,template_updated INTEGER NOT NULL
    ,template_version INTEGER NOT NULL

    -- Ensure unique combination of space ID and UID
    ,UNIQUE (template_space_id, template_uid)

    -- Foreign key to spaces table
    ,CONSTRAINT fk_templates_space_id FOREIGN KEY (template_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE triggers (
    trigger_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,trigger_uid TEXT NOT NULL
    ,trigger_pipeline_id INTEGER NOT NULL
    ,trigger_type TEXT NOT NULL
    ,trigger_repo_id INTEGER NOT NULL
    ,trigger_secret TEXT NOT NULL
    ,trigger_description TEXT NOT NULL
    ,trigger_disabled BOOLEAN NOT NULL
    ,trigger_created_by INTEGER NOT NULL
    ,trigger_actions TEXT NOT NULL
    ,trigger_created INTEGER NOT NULL
    ,trigger_updated INTEGER NOT NULL
    ,trigger_version INTEGER NOT NULL

    -- Ensure unique combination of pipeline ID and UID
    ,UNIQUE (trigger_pipeline_id, trigger_uid)

    -- Foreign key to pipelines table
    ,CONSTRAINT fk_triggers_pipeline_id FOREIGN KEY (trigger_pipeline_id)
        REFERENCES pipelines (pipeline_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE

    -- Foreign key to repositories table
    ,CONSTRAINT fk_triggers_repo_id FOREIGN KEY (trigger_repo_id)
        REFERENCES repositories (repo_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE plugins (
    plugin_uid TEXT NOT NULL
    ,plugin_description TEXT NOT NULL
    ,plugin_logo TEXT NOT NULL
    ,plugin_spec BLOB NOT NULL

    -- Ensure unique plugin names
    ,UNIQUE(plugin_uid)
);