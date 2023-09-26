DROP INDEX jobs_last_executed;
CREATE INDEX jobs_last_executed
    ON jobs(job_last_executed)
    WHERE job_state = 'finished' OR job_state = 'failed';
