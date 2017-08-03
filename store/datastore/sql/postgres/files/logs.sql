-- name: logs-find-proc

SELECT
 log_id
,log_job_id
,log_data
FROM logs
WHERE log_job_id = $1
LIMIT 1
