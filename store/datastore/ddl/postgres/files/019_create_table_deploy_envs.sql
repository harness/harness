-- name: create-table-deploy_envs

CREATE TABLE IF NOT EXISTS deploy_envs (
 deploy_env_id         SERIAL PRIMARY KEY
,deploy_env_build_id   INTEGER
,deploy_env_name       VARCHAR(250)
);

-- name: create-index-deploy_envs-build

CREATE INDEX IF NOT EXISTS deploy_env_build_ix ON deploy_envs (deploy_env_build_id);
