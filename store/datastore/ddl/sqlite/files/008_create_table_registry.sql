-- name: create-table-registry

CREATE TABLE IF NOT EXISTS registry (
 registry_id        INTEGER PRIMARY KEY AUTOINCREMENT
,registry_repo_id   INTEGER
,registry_addr      TEXT
,registry_username  TEXT
,registry_password  TEXT
,registry_email     TEXT
,registry_token     TEXT

,UNIQUE(registry_addr, registry_repo_id)
);

-- name: create-index-registry-repo

CREATE INDEX IF NOT EXISTS ix_registry_repo ON registry (registry_repo_id);
