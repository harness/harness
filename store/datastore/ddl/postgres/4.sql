-- +migrate Up

ALTER TABLE jobs ADD COLUMN job_error VARCHAR(500);

UPDATE jobs SET job_error = '';

-- +migrate Down

ALTER TABLE jobs DROP COLUMN job_error;
