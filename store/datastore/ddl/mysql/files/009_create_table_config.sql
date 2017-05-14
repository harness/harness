-- name: create-table-config

CREATE TABLE IF NOT EXISTS config (
 config_id       INTEGER PRIMARY KEY AUTO_INCREMENT
,config_repo_id  INTEGER
,config_hash     VARCHAR(250)
,config_data     MEDIUMBLOB

,UNIQUE(config_hash, config_repo_id)
);
