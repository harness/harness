-- +migrate Up

CREATE TABLE tasks (
 task_id     TEXT PRIMARY KEY
,task_data   BLOB
,task_labels BLOB
);

-- +migrate Down

DROP TABLE tasks;
