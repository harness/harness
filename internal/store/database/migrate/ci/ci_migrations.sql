DROP TABLE pipelines;
DROP TABLE executions;
DROP TABLE stages;
DROP TABLE steps;
DROP TABLE logs;
CREATE TABLE IF NOT EXISTS pipelines (
    pipeline_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,pipeline_description TEXT NOT NULL
    ,pipeline_space_id INTEGER NOT NULL
    ,pipeline_uid TEXT NOT NULL
    ,pipeline_seq INTEGER NOT NULL DEFAULT 0
    ,pipeline_repo_id INTEGER NOT NULL
    ,pipeline_repo_type TEXT NOT NULL
    ,pipeline_repo_name TEXT NOT NULL
    ,pipeline_default_branch TEXT NOT NULL
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
);

CREATE TABLE IF NOT EXISTS executions (
    execution_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,execution_pipeline_id INTEGER NOT NULL
    ,execution_repo_id INTEGER NOT NULL
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


CREATE TABLE IF NOT EXISTS logs (
    log_id INTEGER PRIMARY KEY
    ,log_data BLOB NOT NULL

    -- Foreign key to steps table
    ,CONSTRAINT fk_logs_id FOREIGN KEY (log_id)
        REFERENCES steps (step_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

-- Insert some pipelines
INSERT INTO pipelines (
    pipeline_id, pipeline_description, pipeline_space_id, pipeline_uid, pipeline_seq,
    pipeline_repo_id, pipeline_repo_type, pipeline_repo_name, pipeline_default_branch,
    pipeline_config_path, pipeline_created, pipeline_updated, pipeline_version
) VALUES (
    1, 'Sample Pipeline 1', 1, 'pipeline_uid_1', 2, 1, 'git', 'sample_repo_1',
    'main', 'config_path_1', 1678932000, 1678932100, 1
);

INSERT INTO pipelines (
    pipeline_id, pipeline_description, pipeline_space_id, pipeline_uid, pipeline_seq,
    pipeline_repo_id, pipeline_repo_type, pipeline_repo_name, pipeline_default_branch,
    pipeline_config_path, pipeline_created, pipeline_updated, pipeline_version
) VALUES (
    2, 'Sample Pipeline 2', 1, 'pipeline_uid_2', 0, 1, 'git', 'sample_repo_2',
    'develop', 'config_path_2', 1678932200, 1678932300, 1
);

-- Insert some executions
INSERT INTO executions (
    execution_id, execution_pipeline_id, execution_repo_id, execution_trigger,
    execution_number, execution_parent, execution_status, execution_error,
    execution_event, execution_action, execution_link, execution_timestamp,
    execution_title, execution_message, execution_before, execution_after,
    execution_ref, execution_source_repo, execution_source, execution_target,
    execution_author, execution_author_name, execution_author_email,
    execution_author_avatar, execution_sender, execution_params, execution_cron,
    execution_deploy, execution_deploy_id, execution_debug, execution_started,
    execution_finished, execution_created, execution_updated, execution_version
) VALUES (
    1, 1, 1, 'manual', 1, 0, 'running', '', 'push', 'created', 
    'https://example.com/pipelines/1', 1678932400, 'Pipeline Execution 1', 
    'Pipeline execution message...', 'commit_hash_before', 'commit_hash_after', 
    'refs/heads/main', 'source_repo_name', 'source_branch', 'target_branch', 
    'author_login', 'Author Name', 'author@example.com', 'https://example.com/avatar.jpg', 
    'sender_username', '{"param1": "value1", "param2": "value2"}', '0 0 * * *', 
    'production', 5, 0, 1678932500, 1678932600, 1678932700, 1678932800, 1
);

INSERT INTO executions (
    execution_id, execution_pipeline_id, execution_repo_id, execution_trigger,
    execution_number, execution_parent, execution_status, execution_error,
    execution_event, execution_action, execution_link, execution_timestamp,
    execution_title, execution_message, execution_before, execution_after,
    execution_ref, execution_source_repo, execution_source, execution_target,
    execution_author, execution_author_name, execution_author_email,
    execution_author_avatar, execution_sender, execution_params, execution_cron,
    execution_deploy, execution_deploy_id, execution_debug, execution_started,
    execution_finished, execution_created, execution_updated, execution_version
) VALUES (
    2, 1, 1, 'manual', 2, 0, 'running', '', 'push', 'created', 
    'https://example.com/pipelines/1', 1678932400, 'Pipeline Execution 1', 
    'Pipeline execution message...', 'commit_hash_before', 'commit_hash_after', 
    'refs/heads/main', 'source_repo_name', 'source_branch', 'target_branch', 
    'author_login', 'Author Name', 'author@example.com', 'https://example.com/avatar.jpg', 
    'sender_username', '{"param1": "value1", "param2": "value2"}', '0 0 * * *', 
    'production', 5, 0, 1678932500, 1678932600, 1678932700, 1678932800, 1
);

INSERT INTO executions (
    execution_id, execution_pipeline_id, execution_repo_id, execution_trigger,
    execution_number, execution_parent, execution_status, execution_error,
    execution_event, execution_action, execution_link, execution_timestamp,
    execution_title, execution_message, execution_before, execution_after,
    execution_ref, execution_source_repo, execution_source, execution_target,
    execution_author, execution_author_name, execution_author_email,
    execution_author_avatar, execution_sender, execution_params, execution_cron,
    execution_deploy, execution_deploy_id, execution_debug, execution_started,
    execution_finished, execution_created, execution_updated, execution_version
) VALUES (
    3, 2, 1, 'manual', 1, 0, 'running', '', 'push', 'created', 
    'https://example.com/pipelines/1', 1678932400, 'Pipeline Execution 1', 
    'Pipeline execution message...', 'commit_hash_before', 'commit_hash_after', 
    'refs/heads/main', 'source_repo_name', 'source_branch', 'target_branch', 
    'author_login', 'Author Name', 'author@example.com', 'https://example.com/avatar.jpg', 
    'sender_username', '{"param1": "value1", "param2": "value2"}', '0 0 * * *', 
    'production', 5, 0, 1678932500, 1678932600, 1678932700, 1678932800, 1
);

INSERT INTO executions (
    execution_id, execution_pipeline_id, execution_repo_id, execution_trigger,
    execution_number, execution_parent, execution_status, execution_error,
    execution_event, execution_action, execution_link, execution_timestamp,
    execution_title, execution_message, execution_before, execution_after,
    execution_ref, execution_source_repo, execution_source, execution_target,
    execution_author, execution_author_name, execution_author_email,
    execution_author_avatar, execution_sender, execution_params, execution_cron,
    execution_deploy, execution_deploy_id, execution_debug, execution_started,
    execution_finished, execution_created, execution_updated, execution_version
) VALUES (
    4, 2, 1, 'manual', 2, 0, 'running', '', 'push', 'created', 
    'https://example.com/pipelines/1', 1678932400, 'Pipeline Execution 1', 
    'Pipeline execution message...', 'commit_hash_before', 'commit_hash_after', 
    'refs/heads/main', 'source_repo_name', 'source_branch', 'target_branch', 
    'author_login', 'Author Name', 'author@example.com', 'https://example.com/avatar.jpg', 
    'sender_username', '{"param1": "value1", "param2": "value2"}', '0 0 * * *', 
    'production', 5, 0, 1678932500, 1678932600, 1678932700, 1678932800, 1
);

-- Insert some stages
INSERT INTO stages (stage_id, stage_execution_id, stage_number, stage_kind, stage_type, stage_name, stage_status, stage_error, stage_errignore, stage_exit_code, stage_limit, stage_os, stage_arch, stage_variant, stage_kernel, stage_machine, stage_started, stage_stopped, stage_created, stage_updated, stage_version, stage_on_success, stage_on_failure, stage_depends_on, stage_labels, stage_limit_repo)
VALUES (
    1,                        -- stage_id
    1,                        -- stage_execution_id
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
    '[]',                       -- stage_depends_on
    '{}',          -- stage_labels
    1                         -- stage_limit_repo
);

-- Sample 2
INSERT INTO stages (stage_id, stage_execution_id, stage_number, stage_kind, stage_type, stage_name, stage_status, stage_error, stage_errignore, stage_exit_code, stage_limit, stage_os, stage_arch, stage_variant, stage_kernel, stage_machine, stage_started, stage_stopped, stage_created, stage_updated, stage_version, stage_on_success, stage_on_failure, stage_depends_on, stage_labels, stage_limit_repo)
VALUES (
    2,                        -- stage_id
    1,                        -- stage_execution_id
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
    '[1]',                      -- stage_depends_on (referring to the first stage)
    '{}',          -- stage_labels
    0                         -- stage_limit_repo (using default value)
);

INSERT INTO steps (step_id, step_stage_id, step_number, step_name, step_status, step_error, step_errignore, step_exit_code, step_started, step_stopped, step_version, step_depends_on, step_image, step_detached, step_schema)
VALUES (
    1,                    -- step_id
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
    '[]',                   -- step_depends_on
    'sample_image',       -- step_image
    0,                    -- step_detached
    'sample_schema'       -- step_schema
);

INSERT INTO steps (step_id, step_stage_id, step_number, step_name, step_status, step_error, step_errignore, step_exit_code, step_started, step_stopped, step_version, step_depends_on, step_image, step_detached, step_schema)
VALUES (
    2,                    -- step_id
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
    '[]',                   -- step_depends_on
    'sample_image',       -- step_image
    0,                    -- step_detached
    'sample_schema'       -- step_schema
);

INSERT INTO steps (step_id, step_stage_id, step_number, step_name, step_status, step_error, step_errignore, step_exit_code, step_started, step_stopped, step_version, step_depends_on, step_image, step_detached, step_schema)
VALUES (
    3,                    -- step_id
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
    '[]',                   -- step_depends_on
    'sample_image',       -- step_image
    0,                    -- step_detached
    'sample_schema'       -- step_schema
);

INSERT INTO steps (step_id, step_stage_id, step_number, step_name, step_status, step_error, step_errignore, step_exit_code, step_started, step_stopped, step_version, step_depends_on, step_image, step_detached, step_schema)
VALUES (
    4,                    -- step_id
    2,                    -- step_stage_id
    2,                    -- step_number
    'stage2step2',        -- step_name
    'Success',            -- step_status
    '',                   -- step_error
    0,                    -- step_errignore
    0,                    -- step_exit_code
    0,                    -- step_started
    0,                    -- step_stopped
    1,                    -- step_version
    '[]',                   -- step_depends_on
    'sample_image',       -- step_image
    0,                    -- step_detached
    'sample_schema'       -- step_schema
);

INSERT INTO logs (log_id, log_data) VALUES (1,
'{"pos": 0, "out": "+git init", "time": 0}
{"pos": 0, "out": "echo Hi", "time": 2}');

INSERT INTO logs (log_id, log_data) VALUES (2,
'{"pos": 0, "out": "+git init", "time": 0}
{"pos": 0, "out": "echo Hi", "time": 2}');

INSERT INTO logs (log_id, log_data) VALUES (3,
'{"pos": 0, "out": "+git init", "time": 0}
{"pos": 0, "out": "echo Hi", "time": 2}');

INSERT INTO logs (log_id, log_data) VALUES (4,
'{"pos": 0, "out": "+git init", "time": 0}
{"pos": 0, "out": "echo Hi", "time": 2}');