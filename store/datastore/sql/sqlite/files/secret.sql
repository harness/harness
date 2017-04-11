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
WHERE secret_repo_id = ?

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
WHERE secret_repo_id = ?
  AND secret_name = ?

-- name: secret-delete

DELETE FROM secrets WHERE secret_id = ?
