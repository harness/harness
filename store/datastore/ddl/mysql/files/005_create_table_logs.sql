-- name: create-table-logs

CREATE TABLE IF NOT EXISTS logs (
 log_id     INTEGER PRIMARY KEY AUTO_INCREMENT
,log_job_id INTEGER
,log_data   MEDIUMBLOB

,UNIQUE(log_job_id)
);
