DROP INDEX jobs_last_executed;
CREATE INDEX jobs_last_executed
    ON jobs(job_last_executed)
    WHERE job_is_recurring = FALSE AND (job_state = 'finished' OR job_state = 'failed' OR job_state = 'canceled');
