ALTER TABLE repositories
    ADD COLUMN repo_importing BOOLEAN DEFAULT FALSE;

UPDATE repositories SET repo_importing = TRUE WHERE repo_state = 1;

ALTER TABLE repositories
    DROP COLUMN repo_state;
