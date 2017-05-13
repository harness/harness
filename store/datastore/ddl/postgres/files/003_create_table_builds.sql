-- name: create-table-builds

CREATE TABLE IF NOT EXISTS builds (
 build_id        SERIAL PRIMARY KEY
,build_repo_id   INTEGER
,build_number    INTEGER
,build_event     VARCHAR(500)
,build_status    VARCHAR(500)
,build_enqueued  INTEGER
,build_created   INTEGER
,build_started   INTEGER
,build_finished  INTEGER
,build_commit    VARCHAR(500)
,build_branch    VARCHAR(500)
,build_ref       VARCHAR(500)
,build_refspec   VARCHAR(1000)
,build_remote    VARCHAR(500)
,build_title     VARCHAR(1000)
,build_message   VARCHAR(2000)
,build_timestamp INTEGER
,build_author    VARCHAR(500)
,build_avatar    VARCHAR(1000)
,build_email     VARCHAR(500)
,build_link      VARCHAR(1000)
,build_deploy    VARCHAR(500)
,build_signed    BOOLEAN
,build_verified  BOOLEAN
,build_parent    INTEGER
,build_error     VARCHAR(500)
,build_reviewer  VARCHAR(250)
,build_reviewed  INTEGER
,build_sender    VARCHAR(250)
,build_config_id INTEGER

,UNIQUE(build_number, build_repo_id)
);

-- name: create-index-builds-repo

CREATE INDEX IF NOT EXISTS ix_build_repo ON builds (build_repo_id);

-- name: create-index-builds-author

CREATE INDEX IF NOT EXISTS ix_build_author ON builds (build_author);
