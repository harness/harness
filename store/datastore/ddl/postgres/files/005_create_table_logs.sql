-- name: create-table-logs

CREATE TABLE IF NOT EXISTS logs (
 log_id     SERIAL PRIMARY KEY
,log_job_id INTEGER
,log_data   BYTEA

,UNIQUE(log_job_id)
);
