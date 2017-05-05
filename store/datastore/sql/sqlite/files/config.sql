-- name: config-find-id

SELECT
 config_id
,config_repo_id
,config_hash
,config_data
FROM config
WHERE config_id = ?

-- name: config-find-repo-hash

SELECT
 config_id
,config_repo_id
,config_hash
,config_data
FROM config
WHERE config_repo_id = ?
  AND config_hash    = ?

-- name: config-find-approved

SELECT build_id FROM builds
WHERE build_repo_id = ?
AND build_config_id = ?
AND build_status NOT IN ('blocked', 'pending')
LIMIT 1
