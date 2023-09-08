ALTER TABLE repositories ADD COLUMN repo_importing BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE repositories ADD COLUMN repo_importing_job_uid TEXT;
