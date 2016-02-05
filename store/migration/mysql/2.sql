-- +migrate Up

ALTER TABLE repos  ADD COLUMN repo_scm     VARCHAR(25);
ALTER TABLE builds ADD COLUMN build_deploy VARCHAR(500);

UPDATE repos  SET repo_scm = 'git' WHERE repo_scm = null;
UPDATE builds SET build_deploy = '' WHERE build_deploy = null;

-- +migrate Down

ALTER TABLE repos  DROP COLUMN repo_scm;
ALTER TABLE builds DROP COLUMN build_deploy;
