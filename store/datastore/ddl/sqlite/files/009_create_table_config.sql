-- name: create-table-config

CREATE TABLE IF NOT EXISTS config (
 config_id       INTEGER PRIMARY KEY AUTOINCREMENT
,config_repo_id  INTEGER
,config_hash     TEXT
,config_data     BLOB
,UNIQUE(config_hash, config_repo_id)
);
