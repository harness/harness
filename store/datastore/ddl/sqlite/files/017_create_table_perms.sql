-- name: create-table-perms

CREATE TABLE IF NOT EXISTS perms (
 perm_user_id INTEGER NOT NULL
,perm_repo_id INTEGER NOT NULL
,perm_pull    BOOLEAN
,perm_push    BOOLEAN
,perm_admin   BOOLEAN
,perm_synced  INTEGER
,UNIQUE(perm_user_id, perm_repo_id)
);

-- name: create-index-perms-repo

CREATE INDEX IF NOT EXISTS ix_perms_repo ON perms (perm_repo_id);

-- name: create-index-perms-user

CREATE INDEX IF NOT EXISTS ix_perms_user ON perms (perm_user_id);
