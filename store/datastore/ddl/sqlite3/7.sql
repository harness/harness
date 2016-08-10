-- +migrate Up

CREATE TABLE team_secrets (
 team_secret_id     INTEGER PRIMARY KEY AUTOINCREMENT
,team_secret_key    TEXT
,team_secret_name   TEXT
,team_secret_value  TEXT
,team_secret_images TEXT
,team_secret_events TEXT

,UNIQUE(team_secret_name, team_secret_key)
);

CREATE INDEX ix_team_secrets_key  ON team_secrets  (team_secret_key);

-- +migrate Down

DROP INDEX ix_team_secrets_key;
DROP TABLE team_secrets;
