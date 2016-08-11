-- +migrate Up

CREATE TABLE team_secrets (
 team_secret_id     INTEGER PRIMARY KEY AUTO_INCREMENT
,team_secret_key    VARCHAR(255)
,team_secret_name   VARCHAR(255)
,team_secret_value  MEDIUMBLOB
,team_secret_images VARCHAR(2000)
,team_secret_events VARCHAR(2000)

,UNIQUE(team_secret_name, team_secret_key)
);

CREATE INDEX ix_team_secrets_key  ON team_secrets  (team_secret_key);

-- +migrate Down

DROP INDEX ix_team_secrets_key;
DROP TABLE team_secrets;
