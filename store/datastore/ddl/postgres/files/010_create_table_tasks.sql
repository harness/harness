-- name: create-table-tasks

CREATE TABLE IF NOT EXISTS tasks (
 task_id     VARCHAR(250) PRIMARY KEY
,task_data   BYTEA
,task_labels BYTEA
);
