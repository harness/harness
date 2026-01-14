ALTER TABLE repositories
    ADD COLUMN repo_importing_job_uid TEXT,
    ADD CONSTRAINT fk_repo_importing_job_uid
        FOREIGN KEY (repo_importing_job_uid)
        REFERENCES jobs(job_uid)
        ON DELETE SET NULL
        ON UPDATE NO ACTION;
