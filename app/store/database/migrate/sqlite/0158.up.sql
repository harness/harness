DROP INDEX IF EXISTS idx_registry_tasks_kind;
DROP INDEX IF EXISTS idx_registry_tasks_created_by;

ALTER TABLE registry_tasks DROP COLUMN registry_task_created_by;
ALTER TABLE registry_tasks DROP COLUMN registry_task_output;
