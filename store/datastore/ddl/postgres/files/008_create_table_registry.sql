-- name: create-table-registry

CREATE TABLE IF NOT EXISTS registry (
 registry_id        SERIAL PRIMARY KEY
,registry_repo_id   INTEGER
,registry_addr      VARCHAR(250)
,registry_email     VARCHAR(500)
,registry_username  VARCHAR(2000)
,registry_password  VARCHAR(8000)
,registry_token     VARCHAR(2000)

,UNIQUE(registry_addr, registry_repo_id)
);

-- name: create-index-registry-repo

CREATE INDEX IF NOT EXISTS ix_registry_repo ON registry (registry_repo_id);
