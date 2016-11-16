-- +migrate Up

ALTER TABLE secrets      ADD COLUMN secret_conceal      BOOLEAN;
ALTER TABLE team_secrets ADD COLUMN team_secret_conceal BOOLEAN;

UPDATE secrets      SET secret_conceal      = false;
UPDATE team_secrets SET team_secret_conceal = false;

-- +migrate Down

ALTER TABLE secrets      DROP COLUMN secret_conceal;
ALTER TABLE team_secrets DROP COLUMN team_secret_conceal;
