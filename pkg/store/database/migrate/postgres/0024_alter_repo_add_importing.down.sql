ALTER TABLE repositories
    DROP CONSTRAINT fk_repo_importing_job_uid,
    DROP COLUMN repo_importing_job_uid,
    DROP COLUMN repo_importing;
