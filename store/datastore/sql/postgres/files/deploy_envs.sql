-- name: deploy_envs-find-build

SELECT
 deploy_env_id
,deploy_env_build_id
,deploy_env_name
FROM deploy_envs
WHERE deploy_env_build_id = $1
ORDER BY deploy_env_id ASC

