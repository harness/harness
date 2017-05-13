-- name: create-table-config

CREATE TABLE IF NOT EXISTS config (
 config_id       SERIAL PRIMARY KEY
,config_repo_id  INTEGER
,config_hash     VARCHAR(250)
,config_data     BYTEA

,UNIQUE(config_hash, config_repo_id)
);
