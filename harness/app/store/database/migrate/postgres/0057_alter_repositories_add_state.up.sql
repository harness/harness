ALTER TABLE repositories
    ADD COLUMN repo_state INTEGER DEFAULT 0;

UPDATE repositories SET repo_state = 1 WHERE repo_importing = TRUE;

ALTER TABLE repositories
    DROP COLUMN repo_importing;
