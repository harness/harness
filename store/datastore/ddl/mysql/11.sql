-- +migrate Up

ALTER TABLE builds ADD COLUMN build_error  VARCHAR(500);
UPDATE builds SET build_error = '' WHERE job_error = null;

-- +migrate Down

ALTER TABLE builds DROP COLUMN build_error;
