-- +migrate Up

ALTER TABLE repos  ADD COLUMN repo_config_path VARCHAR(255);
ALTER TABLE builds ADD COLUMN build_sender     VARCHAR(255);
ALTER TABLE builds ADD COLUMN build_reviewer   VARCHAR(255);
ALTER TABLE builds ADD COLUMN build_reviewed   INTEGER;

UPDATE repos  SET repo_config_path = '.drone.yml';
UPDATE builds SET build_reviewer   = '';
UPDATE builds SET build_reviewed   = 0;
UPDATE builds SET build_sender     = '';

-- +migrate Down

ALTER TABLE repos  DROP COLUMN repo_config_path;
ALTER TABLE builds DROP COLUMN build_sender;
ALTER TABLE builds DROP COLUMN build_reviewer;
ALTER TABLE builds DROP COLUMN build_reviewed;
