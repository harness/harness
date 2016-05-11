-- +migrate Up

ALTER TABLE jobs ADD COLUMN job_error VARCHAR(500);

UPDATE jobs SET job_error = '' WHERE job_error = null;

-- +migrate Down

ALTER TABLE jobs DROP COLUMN job_error;
