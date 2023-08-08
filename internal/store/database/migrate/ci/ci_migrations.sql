CREATE TABLE IF NOT EXISTS pipelines (
    pipeline_id INTEGER PRIMARY KEY AUTOINCREMENT,
    pipeline_description TEXT,
    pipeline_parent_id INTEGER NOT NULL,
    pipeline_uid TEXT NOT NULL,
    pipeline_seq INTEGER NOT NULL DEFAULT 0,
    pipeline_repo_id INTEGER,
    pipeline_repo_type TEXT NOT NULL,
    pipeline_repo_name TEXT,
    pipeline_default_branch TEXT,
    pipeline_config_path TEXT NOT NULL,
    pipeline_created INTEGER,
    pipeline_updated INTEGER,
    pipeline_version INTEGER,

    -- Ensure unique combination of UID and ParentID
    UNIQUE (pipeline_parent_id, pipeline_uid),

    -- Foreign key to spaces table
    CONSTRAINT fk_pipeline_parent_id FOREIGN KEY (pipeline_parent_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE

    -- Foreign key to repositories table
    CONSTRAINT fk_pipeline_repo_id FOREIGN KEY (pipeline_repo_id)
        REFERENCES repositories (repo_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS secrets (
    execution_id INTEGER PRIMARY KEY AUTOINCREMENT,
    execution_pipeline_id INTEGER NOT NULL,
    execution_repo_id INTEGER,
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

CREATE TABLE IF NOT EXISTS secrets (
    secret_id INTEGER PRIMARY KEY AUTOINCREMENT,
    secret_uid TEXT NOT NULL,
    secret_parent_id INTEGER NOT NULL,
    secret_description TEXT NOT NULL,
    secret_data BLOB NOT NULL,
    secret_created INTEGER,
    secret_updated INTEGER,
    secret_version INTEGER,

    -- Ensure unique combination of space ID and UID
    UNIQUE (secret_parent_id, secret_uid),

    -- Foreign key to spaces table
    CONSTRAINT fk_secret_parent_id FOREIGN KEY (secret_parent_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);