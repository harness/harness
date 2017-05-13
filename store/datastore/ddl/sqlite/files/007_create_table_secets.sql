-- name: create-table-secrets

CREATE TABLE IF NOT EXISTS secrets (
 secret_id          INTEGER PRIMARY KEY AUTOINCREMENT
,secret_repo_id     INTEGER
,secret_name        TEXT
,secret_value       TEXT
,secret_images      TEXT
,secret_events      TEXT
,secret_skip_verify BOOLEAN
,secret_conceal     BOOLEAN
,UNIQUE(secret_name, secret_repo_id)
);

-- name: create-index-secrets-repo

CREATE INDEX IF NOT EXISTS ix_secrets_repo ON secrets (secret_repo_id);
