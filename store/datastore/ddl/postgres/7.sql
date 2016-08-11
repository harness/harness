-- +migrate Up

CREATE TABLE team_secrets (
 team_secret_id     SERIAL PRIMARY KEY
,team_secret_key    VARCHAR(255)
,team_secret_name   VARCHAR(255)
,team_secret_value  BYTEA
,team_secret_images VARCHAR(2000)
,team_secret_events VARCHAR(2000)

,UNIQUE(team_secret_name, team_secret_key)
);

CREATE INDEX ix_team_secrets_key  ON team_secrets  (team_secret_key);

-- +migrate Down

DROP INDEX ix_team_secrets_key;
DROP TABLE team_secrets;
