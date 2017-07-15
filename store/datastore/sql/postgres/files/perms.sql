-- name: perms-find-user

SELECT
 perm_user_id
,perm_repo_id
,perm_pull
,perm_push
,perm_admin
,perm_date
FROM perms
WHERE perm_user_id = $1

-- name: perms-find-user-repo

SELECT
 perm_user_id
,perm_repo_id
,perm_pull
,perm_push
,perm_admin
,perm_synced
FROM perms
WHERE perm_user_id = $1
  AND perm_repo_id = $2

-- name: perms-insert-replace

REPLACE INTO perms (
 perm_user_id
,perm_repo_id
,perm_pull
,perm_push
,perm_admin
,perm_synced
) VALUES ($1,$2,$3,$4,$5,$6)

-- name: perms-insert-replace-lookup

INSERT INTO perms (
 perm_user_id
,perm_repo_id
,perm_pull
,perm_push
,perm_admin
,perm_synced
) VALUES ($1,(SELECT repo_id FROM repos WHERE repo_full_name = $2),$3,$4,$5,$6)
ON CONFLICT (perm_user_id, perm_repo_id) DO UPDATE SET
 perm_pull = EXCLUDED.perm_pull
,perm_push = EXCLUDED.perm_push
,perm_admin = EXCLUDED.perm_admin
,perm_synced = EXCLUDED.perm_synced

-- name: perms-delete-user-repo

DELETE FROM perms
WHERE perm_user_id = $1
  AND perm_repo_id = $2

-- name: perms-delete-user-date

DELETE FROM perms
WHERE perm_user_id = $1
  AND perm_synced < $2
