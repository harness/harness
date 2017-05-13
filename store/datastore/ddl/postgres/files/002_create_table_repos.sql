-- name: create-table-repos

CREATE TABLE IF NOT EXISTS repos (
 repo_id            SERIAL PRIMARY KEY
,repo_user_id       INTEGER
,repo_owner         VARCHAR(250)
,repo_name          VARCHAR(250)
,repo_full_name     VARCHAR(250)
,repo_avatar        VARCHAR(500)
,repo_link          VARCHAR(1000)
,repo_clone         VARCHAR(1000)
,repo_branch        VARCHAR(500)
,repo_timeout       INTEGER
,repo_private       BOOLEAN
,repo_trusted       BOOLEAN
,repo_allow_pr      BOOLEAN
,repo_allow_push    BOOLEAN
,repo_allow_deploys BOOLEAN
,repo_allow_tags    BOOLEAN
,repo_hash          VARCHAR(500)
,repo_scm           VARCHAR(50)
,repo_config_path   VARCHAR(500)
,repo_gated         BOOLEAN

,UNIQUE(repo_full_name)
);
