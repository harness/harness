-- name: create-table-logs

CREATE TABLE IF NOT EXISTS logs (
 log_id     INTEGER PRIMARY KEY AUTOINCREMENT
,log_job_id INTEGER
,log_data   BLOB
,UNIQUE(log_job_id)
);
