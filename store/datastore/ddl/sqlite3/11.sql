-- +migrate Up

ALTER TABLE builds ADD COLUMN build_error TEXT;
UPDATE builds SET build_error = '';

-- +migrate Down

ALTER TABLE builds DROP COLUMN build_error;
