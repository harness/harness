-- name: create-table-builds

CREATE TABLE IF NOT EXISTS builds (
 build_id        INTEGER PRIMARY KEY AUTOINCREMENT
,build_repo_id   INTEGER
,build_number    INTEGER
,build_event     TEXT
,build_status    TEXT
,build_enqueued  INTEGER
,build_created   INTEGER
,build_started   INTEGER
,build_finished  INTEGER
,build_commit    TEXT
,build_branch    TEXT
,build_ref       TEXT
,build_refspec   TEXT
,build_remote    TEXT
,build_title     TEXT
,build_message   TEXT
,build_timestamp INTEGER
,build_author    TEXT
,build_avatar    TEXT
,build_email     TEXT
,build_link      TEXT
,build_deploy    TEXT
,build_signed    BOOLEAN
,build_verified  BOOLEAN
,build_parent    INTEGER
,build_error     TEXT
,build_reviewer  TEXT
,build_reviewed  INTEGER
,build_sender    TEXT
,build_config_id INTEGER
,UNIQUE(build_number, build_repo_id)
);

-- name: create-index-builds-repo

CREATE INDEX IF NOT EXISTS ix_build_repo ON builds (build_repo_id);

-- name: create-index-builds-author

CREATE INDEX IF NOT EXISTS ix_build_author ON builds (build_author);

-- name: create-index-builds-status

CREATE INDEX IF NOT EXISTS ix_build_status_running ON builds (build_status)
WHERE build_status IN ('pending', 'running');
