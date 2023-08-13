CREATE TABLE IF NOT EXISTS pipelines (
    pipeline_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,pipeline_description TEXT NOT NULL
    ,pipeline_space_id INTEGER NOT NULL
    ,pipeline_uid TEXT NOT NULL
    ,pipeline_seq INTEGER NOT NULL DEFAULT 0
    ,pipeline_repo_id INTEGER
    ,pipeline_connector_id INTEGER
    ,pipeline_repo_type TEXT NOT NULL
    ,pipeline_repo_name TEXT
    ,pipeline_default_branch TEXT
    ,pipeline_config_path TEXT NOT NULL
    ,pipeline_created INTEGER NOT NULL
    ,pipeline_updated INTEGER NOT NULL
    ,pipeline_version INTEGER NOT NULL

    -- Ensure unique combination of UID and ParentID
    ,UNIQUE (pipeline_space_id, pipeline_uid)

    -- Foreign key to spaces table
    ,CONSTRAINT fk_pipeline_space_id FOREIGN KEY (pipeline_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE

    -- Foreign key to repositories table
    ,CONSTRAINT fk_pipelines_repo_id FOREIGN KEY (pipeline_repo_id)
        REFERENCES repositories (repo_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
    
    -- Foreign key to connectors table
    ,CONSTRAINT fk_pipelines_connector_id FOREIGN KEY (connector_id)
        REFERENCES connectors (connector_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS executions (
    execution_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,execution_pipeline_id INTEGER NOT NULL
    ,execution_repo_id INTEGER
    ,execution_trigger TEXT
    ,execution_number INTEGER NOT NULL
    ,execution_parent INTEGER
    ,execution_status TEXT
    ,execution_error TEXT
    ,execution_event TEXT
    ,execution_action TEXT
    ,execution_link TEXT
    ,execution_timestamp INTEGER
    ,execution_title TEXT
    ,execution_message TEXT
    ,execution_before TEXT
    ,execution_after TEXT
    ,execution_ref TEXT
    ,execution_source_repo TEXT
    ,execution_source TEXT
    ,execution_target TEXT
    ,execution_author TEXT
    ,execution_author_name TEXT
    ,execution_author_email TEXT
    ,execution_author_avatar TEXT
    ,execution_sender TEXT
    ,execution_params TEXT
    ,execution_cron TEXT
    ,execution_deploy TEXT
    ,execution_deploy_id INTEGER
    ,execution_debug BOOLEAN NOT NULL DEFAULT 0
    ,execution_started INTEGER
    ,execution_finished INTEGER
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
);

CREATE TABLE IF NOT EXISTS secrets (
    secret_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,secret_uid TEXT NOT NULL
    ,secret_space_id INTEGER NOT NULL
    ,secret_description TEXT NOT NULL
    ,secret_data BLOB NOT NULL
    ,secret_created INTEGER NOT NULL
    ,secret_updated INTEGER NOT NULL
    ,secret_version INTEGER NOT NULL

    -- Ensure unique combination of space ID and UID
    ,UNIQUE (secret_space_id, secret_uid)

    -- Foreign key to spaces table
    ,CONSTRAINT fk_secrets_space_id FOREIGN KEY (secret_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS stages (
    stage_id          INTEGER PRIMARY KEY AUTOINCREMENT
    ,stage_pipeline_id     INTEGER
    ,stage_execution_id    INTEGER
    ,stage_number      INTEGER
    ,stage_kind        TEXT
    ,stage_type        TEXT
    ,stage_name        TEXT
    ,stage_status      TEXT
    ,stage_error       TEXT
    ,stage_errignore   BOOLEAN
    ,stage_exit_code   INTEGER
    ,stage_limit       INTEGER
    ,stage_os          TEXT
    ,stage_arch        TEXT
    ,stage_variant     TEXT
    ,stage_kernel      TEXT
    ,stage_machine     TEXT
    ,stage_started     INTEGER
    ,stage_stopped     INTEGER
    ,stage_created     INTEGER
    ,stage_updated     INTEGER
    ,stage_version     INTEGER
    ,stage_on_success  BOOLEAN
    ,stage_on_failure  BOOLEAN
    ,stage_depends_on  TEXT
    ,stage_labels      TEXT
    ,stage_limit_repo INTEGER NOT NULL DEFAULT 0

    -- Ensure unique combination of stage execution ID and stage number
    ,UNIQUE(stage_execution_id, stage_number)

    -- Foreign key to executions table
    ,CONSTRAINT fk_stages_execution_id FOREIGN KEY (stage_execution_id)
        REFERENCES executions (execution_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

-- name: create-index-stages-build

CREATE INDEX IF NOT EXISTS ix_stages_build ON stages (stage_execution_id);

-- name: create-index-stages-status

CREATE INDEX IF NOT EXISTS ix_stage_in_progress ON stages (stage_status)
WHERE stage_status IN ('pending', 'running');


CREATE TABLE IF NOT EXISTS steps (
    step_id          INTEGER PRIMARY KEY AUTOINCREMENT
    ,step_stage_id    INTEGER
    ,step_number      INTEGER
    ,step_name        VARCHAR(100)
    ,step_status      VARCHAR(50)
    ,step_error       VARCHAR(500)
    ,step_errignore   BOOLEAN
    ,step_exit_code   INTEGER
    ,step_started     INTEGER
    ,step_stopped     INTEGER
    ,step_version     INTEGER

    -- Ensure unique comination of stage ID and step number
    ,UNIQUE(step_stage_id, step_number)

    -- Foreign key to stages table
    ,CONSTRAINT fk_steps_stage_id FOREIGN KEY (step_stage_id)
        REFERENCES stages (stage_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE 
);

CREATE TABLE IF NOT EXISTS connectors (
    connector_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,connector_type TEXT NOT NULL
    ,connector_uid TEXT NOT NULL
    ,connector_space_id INTEGER NOT NULL
    ,connector_created INTEGER NOT NULL
    ,connector_updated INTEGER NOT NULL
    ,connector_version INTEGER NOT NULL

    -- Foreign key to spaces table
    ,CONSTRAINT fk_connectors_space_id FOREIGN KEY (connector_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE

    -- Ensure unique combination of space ID and connector UID
    ,UNIQUE (connector_space_id, connector_uid)
)


CREATE TABLE IF NOT EXISTS logs (
    log_id INTEGER PRIMARY KEY
    ,log_data BLOB NOT NULL

    -- Foreign key to steps table
    ,CONSTRAINT fk_logs_id FOREIGN KEY (log_id)
        REFERENCES steps (step_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);