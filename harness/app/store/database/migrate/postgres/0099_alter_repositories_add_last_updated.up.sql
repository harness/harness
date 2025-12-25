ALTER TABLE repositories
    ADD COLUMN repo_last_git_push BIGINT NOT NULL DEFAULT 0;

-- backfill an approximation to ensure roughly correct order at time of introduction of this field.
UPDATE repositories
SET repo_last_git_push = repo_updated;