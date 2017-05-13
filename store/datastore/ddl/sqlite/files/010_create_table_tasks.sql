-- name: create-table-tasks

CREATE TABLE IF NOT EXISTS tasks (
 task_id     TEXT PRIMARY KEY
,task_data   BLOB
,task_labels BLOB
);
