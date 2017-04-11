-- name: secret-find-repo

SELECT
 secret_id
,secret_repo_id
,secret_name
,secret_value
,secret_images
,secret_events
,secret_conceal
,secret_skip_verify
FROM secrets
WHERE secret_repo_id = $1

-- name: secret-find-repo-name

SELECT
secret_id
,secret_repo_id
,secret_name
,secret_value
,secret_images
,secret_events
,secret_conceal
,secret_skip_verify
FROM secrets
WHERE secret_repo_id = $1
  AND secret_name = $2

-- name: secret-delete

DELETE FROM secrets WHERE secret_id = $1
