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
    pipeline_config_path TEXT,
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