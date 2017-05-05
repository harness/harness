-- +migrate Up

CREATE TABLE config (
 config_id       SERIAL PRIMARY KEY
,config_repo_id  INTEGER
,config_hash     VARCHAR(250)
,config_data     BYTEA
,config_approved BOOLEAN

,UNIQUE(config_hash, config_repo_id)
);

ALTER TABLE builds ADD COLUMN build_config_id INTEGER;
UPDATE builds set build_config_id = 0;

-- +migrate Down

DROP TABLE config;
