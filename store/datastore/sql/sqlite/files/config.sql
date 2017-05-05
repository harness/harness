-- name: config-find-id

SELECT
 config_id
,config_repo_id
,config_hash
,config_data
,config_approved
FROM config
WHERE config_id = ?

-- name: config-find-repo-hash

SELECT
 config_id
,config_repo_id
,config_hash
,config_data
,config_approved
FROM config
WHERE config_repo_id = ?
  AND config_hash    = ?
