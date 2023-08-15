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
    ,stage_execution_id    INTEGER NOT NULL
    ,stage_number      INTEGER NOT NULL
    ,stage_kind        TEXT NOT NULL
    ,stage_type        TEXT NOT NULL
    ,stage_name        TEXT NOT NULL
    ,stage_status      TEXT NOT NULL
    ,stage_error       TEXT NOT NULL
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

-- name: create-index-stages-build

CREATE INDEX IF NOT EXISTS ix_stages_build ON stages (stage_execution_id);

-- name: create-index-stages-status

CREATE INDEX IF NOT EXISTS ix_stage_in_progress ON stages (stage_status)
WHERE stage_status IN ('pending', 'running');


CREATE TABLE IF NOT EXISTS steps (
    step_id          INTEGER PRIMARY KEY AUTOINCREMENT
    ,step_stage_id    INTEGER NOT NULL
    ,step_number      INTEGER NOT NULL
    ,step_name        VARCHAR(100) NOT NULL
    ,step_status      VARCHAR(50) NOT NULL
    ,step_error       VARCHAR(500) NOT NULL
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
);


CREATE TABLE IF NOT EXISTS logs (
    log_id INTEGER PRIMARY KEY
    ,log_data BLOB NOT NULL

    -- Foreign key to steps table
    ,CONSTRAINT fk_logs_id FOREIGN KEY (log_id)
        REFERENCES steps (step_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

-- Sample 1
INSERT INTO stages (stage_execution_id, stage_number, stage_kind, stage_type, stage_name, stage_status, stage_error, stage_errignore, stage_exit_code, stage_limit, stage_os, stage_arch, stage_variant, stage_kernel, stage_machine, stage_started, stage_stopped, stage_created, stage_updated, stage_version, stage_on_success, stage_on_failure, stage_depends_on, stage_labels, stage_limit_repo)
VALUES (
    3,                        -- stage_execution_id
    1,                        -- stage_number
    'build',                  -- stage_kind
    'docker',                 -- stage_type
    'Build Stage',            -- stage_name
    'Pending',                -- stage_status
    '',                       -- stage_error
    0,                        -- stage_errignore
    0,                        -- stage_exit_code
    2,                        -- stage_limit
    'linux',                  -- stage_os
    'x86_64',                 -- stage_arch
    'default',                -- stage_variant
    '4.18.0-305.7.1.el8_4.x86_64',  -- stage_kernel
    'x86_64',                 -- stage_machine
    0,                        -- stage_started
    0,                        -- stage_stopped
    1679089460,               -- stage_created
    1679089500,               -- stage_updated
    1,                        -- stage_version
    1,                        -- stage_on_success
    0,                        -- stage_on_failure
    '',                       -- stage_depends_on
    'label1,label2',          -- stage_labels
    1                         -- stage_limit_repo
);

-- Sample 2
INSERT INTO stages (stage_execution_id, stage_number, stage_kind, stage_type, stage_name, stage_status, stage_error, stage_errignore, stage_exit_code, stage_limit, stage_os, stage_arch, stage_variant, stage_kernel, stage_machine, stage_started, stage_stopped, stage_created, stage_updated, stage_version, stage_on_success, stage_on_failure, stage_depends_on, stage_labels, stage_limit_repo)
VALUES (
    3,                        -- stage_execution_id
    2,                        -- stage_number
    'test',                   -- stage_kind
    'pytest',                 -- stage_type
    'Test Stage',             -- stage_name
    'Pending',                -- stage_status
    '',                       -- stage_error
    0,                        -- stage_errignore
    0,                        -- stage_exit_code
    1,                        -- stage_limit
    'linux',                  -- stage_os
    'x86_64',                 -- stage_arch
    'default',                -- stage_variant
    '4.18.0-305.7.1.el8_4.x86_64',  -- stage_kernel
    'x86_64',                 -- stage_machine
    0,                        -- stage_started
    0,                        -- stage_stopped
    1679089560,               -- stage_created
    1679089600,               -- stage_updated
    1,                        -- stage_version
    1,                        -- stage_on_success
    1,                        -- stage_on_failure
    '1',                      -- stage_depends_on (referring to the first stage)
    'label3,label4',          -- stage_labels
    0                         -- stage_limit_repo (using default value)
);

INSERT INTO steps (step_stage_id, step_number, step_name, step_status, step_error, step_errignore, step_exit_code, step_started, step_stopped, step_version, step_depends_on, step_image, step_detached, step_schema)
VALUES (
    1,                    -- step_stage_id
    1,                    -- step_number
    'stage1step1',        -- step_name
    'Pending',            -- step_status
    '',                   -- step_error
    0,                    -- step_errignore
    0,                    -- step_exit_code
    0,                    -- step_started
    0,                    -- step_stopped
    1,                    -- step_version
    '',                   -- step_depends_on
    'sample_image',       -- step_image
    0,                    -- step_detached
    'sample_schema'       -- step_schema
);

INSERT INTO steps (step_stage_id, step_number, step_name, step_status, step_error, step_errignore, step_exit_code, step_started, step_stopped, step_version, step_depends_on, step_image, step_detached, step_schema)
VALUES (
    1,                    -- step_stage_id
    2,                    -- step_number
    'stage1step2',        -- step_name
    'Success',            -- step_status
    '',                   -- step_error
    0,                    -- step_errignore
    0,                    -- step_exit_code
    0,                    -- step_started
    0,                    -- step_stopped
    1,                    -- step_version
    '',                   -- step_depends_on
    'sample_image',       -- step_image
    0,                    -- step_detached
    'sample_schema'       -- step_schema
);

INSERT INTO steps (step_stage_id, step_number, step_name, step_status, step_error, step_errignore, step_exit_code, step_started, step_stopped, step_version, step_depends_on, step_image, step_detached, step_schema)
VALUES (
    2,                    -- step_stage_id
    1,                    -- step_number
    'stage2step1',        -- step_name
    'Success',            -- step_status
    '',                   -- step_error
    0,                    -- step_errignore
    0,                    -- step_exit_code
    0,                    -- step_started
    0,                    -- step_stopped
    1,                    -- step_version
    '',                   -- step_depends_on
    'sample_image',       -- step_image
    0,                    -- step_detached
    'sample_schema'       -- step_schema
);

INSERT INTO steps (step_stage_id, step_number, step_name, step_status, step_error, step_errignore, step_exit_code, step_started, step_stopped, step_version, step_depends_on, step_image, step_detached, step_schema)
VALUES (
    2,                    -- step_stage_id
    1,                    -- step_number
    'stage2step2',        -- step_name
    'Success',            -- step_status
    '',                   -- step_error
    0,                    -- step_errignore
    0,                    -- step_exit_code
    0,                    -- step_started
    0,                    -- step_stopped
    1,                    -- step_version
    '',                   -- step_depends_on
    'sample_image',       -- step_image
    0,                    -- step_detached
    'sample_schema'       -- step_schema
);


INSERT INTO steps (step_stage_id, step_number, step_name) VALUES (1, 1, "step1");
INSERT INTO steps (step_stage_id, step_number, step_name) VALUES (1, 2, "step2");

INSERT INTO steps (step_stage_id, step_number, step_name) VALUES (2, 1, "step1");
INSERT INTO steps (step_stage_id, step_number, step_name) VALUES (2, 2, "step2");

INSERT INTO logs (log_id, log_data) VALUES (1, "stage1 step1 logs");
INSERT INTO logs (log_id, log_data) VALUES (2, "stage1 step2 logs");
INSERT INTO logs (log_id, log_data) VALUES (3, "stage2 step1 logs");
INSERT INTO logs (log_id, log_data) VALUES (4, "stage2 step2 logs");