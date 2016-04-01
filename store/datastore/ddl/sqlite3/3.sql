-- +migrate Up

CREATE TABLE secrets (
 secret_id        INTEGER PRIMARY KEY AUTOINCREMENT
,secret_repo_id   INTEGER
,secret_name      TEXT
,secret_value     TEXT
,secret_images    TEXT
,secret_events    TEXT

,UNIQUE(secret_name, secret_repo_id)
);

CREATE TABLE registry (
 registry_id        INTEGER PRIMARY KEY AUTOINCREMENT
,registry_repo_id   INTEGER
,registry_addr      TEXT
,registry_username  TEXT
,registry_password  TEXT
,registry_email     TEXT
,registry_token     TEXT

,UNIQUE(registry_addr, registry_repo_id)
);

CREATE INDEX ix_secrets_repo  ON secrets  (secret_repo_id);
CREATE INDEX ix_registry_repo ON registry (registry_repo_id);

-- +migrate Down

DROP INDEX ix_secrets_repo;
DROP INDEX ix_registry_repo;
DROP TABLE secrets;
DROP TABLE registry;
