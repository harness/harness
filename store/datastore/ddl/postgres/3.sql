-- +migrate Up

CREATE TABLE secrets (
 secret_id        SERIAL PRIMARY KEY
,secret_repo_id   INTEGER
,secret_name      VARCHAR(500)
,secret_value     BYTEA
,secret_images    VARCHAR(2000)
,secret_events    VARCHAR(2000)

,UNIQUE(secret_name, secret_repo_id)
);

CREATE TABLE registry (
 registry_id        SERIAL PRIMARY KEY
,registry_repo_id   INTEGER
,registry_addr      VARCHAR(500)
,registry_email     VARCHAR(500)
,registry_username  VARCHAR(2000)
,registry_password  VARCHAR(2000)
,registry_token     VARCHAR(2000)

,UNIQUE(registry_addr, registry_repo_id)
);

CREATE INDEX ix_secrets_repo  ON secrets  (secret_repo_id);
CREATE INDEX ix_registry_repo ON registry (registry_repo_id);

-- +migrate Down

DROP INDEX ix_secrets_repo;
DROP INDEX ix_registry_repo;
