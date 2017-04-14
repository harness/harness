-- name: task-list

SELECT
 task_id
,task_data
,task_labels
FROM tasks

-- name: task-delete

DELETE FROM tasks WHERE task_id = $1
