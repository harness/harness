-- +migrate Up

CREATE TABLE tasks (
 task_id     VARCHAR(255) PRIMARY KEY
,task_data   MEDIUMBLOB
,task_labels MEDIUMBLOB
);

-- +migrate Down

DROP TABLE tasks;
