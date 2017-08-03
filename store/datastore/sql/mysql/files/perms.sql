-- name: perms-find-user

SELECT
 perm_user_id
,perm_repo_id
,perm_pull
,perm_push
,perm_admin
,perm_date
FROM perms
WHERE perm_user_id = ?

-- name: perms-find-user-repo

SELECT
 perm_user_id
,perm_repo_id
,perm_pull
,perm_push
,perm_admin
,perm_synced
FROM perms
WHERE perm_user_id = ?
  AND perm_repo_id = ?

-- name: perms-insert-replace

REPLACE INTO perms (
 perm_user_id
,perm_repo_id
,perm_pull
,perm_push
,perm_admin
,perm_synced
) VALUES (?,?,?,?,?,?)

-- name: perms-insert-replace-lookup

REPLACE INTO perms (
 perm_user_id
,perm_repo_id
,perm_pull
,perm_push
,perm_admin
,perm_synced
) VALUES (?,(SELECT repo_id FROM repos WHERE repo_full_name = ?),?,?,?,?)

-- name: perms-delete-user-repo

DELETE FROM perms
WHERE perm_user_id = ?
  AND perm_repo_id = ?

-- name: perms-delete-user-date

DELETE FROM perms
WHERE perm_user_id = ?
  AND perm_synced < ?
