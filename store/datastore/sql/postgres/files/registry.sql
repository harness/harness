-- name: registry-find-repo

SELECT
 registry_id
,registry_repo_id
,registry_addr
,registry_username
,registry_password
,registry_email
,registry_token
FROM registry
WHERE registry_repo_id = $1

-- name: registry-find-repo-addr

SELECT
 registry_id
,registry_repo_id
,registry_addr
,registry_username
,registry_password
,registry_email
,registry_token
FROM registry
WHERE registry_repo_id = $1
  AND registry_addr = $2

-- name: registry-delete-repo

DELETE FROM registry WHERE registry_repo_id = $1

-- name: registry-delete

DELETE FROM registry WHERE registry_id = $1
