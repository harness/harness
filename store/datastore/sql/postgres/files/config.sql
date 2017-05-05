-- name: config-find-id

SELECT
 config_id
,config_repo_id
,config_hash
,config_data
FROM config
WHERE config_id = $1

-- name: config-find-repo-hash

SELECT
 config_id
,config_repo_id
,config_hash
,config_data
FROM config
WHERE config_repo_id = $1
  AND config_hash    = $2

-- name: config-find-approved

SELECT build_id FROM builds
WHERE build_repo_id = $1
AND build_config_id = $2
AND build_status NOT IN ('blocked', 'pending')
LIMIT 1
