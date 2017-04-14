-- +migrate Up

CREATE TABLE tasks (
 task_id     VARCHAR(255) PRIMARY KEY
,task_data   BYTEA
,task_labels BYTEA
);

-- +migrate Down

DROP TABLE tasks;
