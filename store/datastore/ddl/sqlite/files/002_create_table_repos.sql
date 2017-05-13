-- name: create-table-repos

CREATE TABLE IF NOT EXISTS repos (
 repo_id            INTEGER PRIMARY KEY AUTOINCREMENT
,repo_user_id       INTEGER
,repo_owner         TEXT
,repo_name          TEXT
,repo_full_name     TEXT
,repo_avatar        TEXT
,repo_link          TEXT
,repo_clone         TEXT
,repo_branch        TEXT
,repo_timeout       INTEGER
,repo_private       BOOLEAN
,repo_trusted       BOOLEAN
,repo_allow_pr      BOOLEAN
,repo_allow_push    BOOLEAN
,repo_allow_deploys BOOLEAN
,repo_allow_tags    BOOLEAN
,repo_hash          TEXT
,repo_scm           TEXT
,repo_config_path   TEXT
,repo_gated         BOOLEAN
,UNIQUE(repo_full_name)
);
