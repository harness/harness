-- name: create-table-deploy_envs

CREATE TABLE IF NOT EXISTS deploy_envs (
 deploy_env_id         INTEGER PRIMARY KEY AUTOINCREMENT
,deploy_env_build_id   INTEGER
,deploy_env_name       TEXT
);

-- name: create-index-deploy_envs-build

CREATE INDEX IF NOT EXISTS deploy_env_build_ix ON deploy_envs (deploy_env_build_id);
