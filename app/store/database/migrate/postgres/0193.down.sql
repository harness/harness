DROP INDEX task_links_kind_ref_id;
DROP TABLE task_links;
DROP TABLE task_labels;
DROP INDEX task_activities_task_id_created;
DROP TABLE task_activities;
DROP TABLE tasks;
ALTER TABLE repositories DROP COLUMN repo_task_seq;
