-- name: create-table-secrets

CREATE TABLE IF NOT EXISTS secrets (
 secret_id          SERIAL PRIMARY KEY
,secret_repo_id     INTEGER
,secret_name        VARCHAR(250)
,secret_value       BYTEA
,secret_images      VARCHAR(2000)
,secret_events      VARCHAR(2000)
,secret_skip_verify BOOLEAN
,secret_conceal     BOOLEAN

,UNIQUE(secret_name, secret_repo_id)
);

-- name: create-index-secrets-repo

CREATE INDEX IF NOT EXISTS ix_secrets_repo  ON secrets  (secret_repo_id);
