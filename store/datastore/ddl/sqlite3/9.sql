-- +migrate Up

ALTER TABLE secrets      ADD COLUMN secret_skip_verify      BOOLEAN;
ALTER TABLE team_secrets ADD COLUMN team_secret_skip_verify BOOLEAN;

UPDATE secrets      SET secret_skip_verify      = 0;
UPDATE team_secrets SET team_secret_skip_verify = 0;

-- +migrate Down

ALTER TABLE secrets      DROP COLUMN secret_skip_verify;
ALTER TABLE team_secrets DROP COLUMN team_secret_skip_verify;
