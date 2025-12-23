ALTER TABLE registry_tasks ADD COLUMN registry_task_output text;
ALTER TABLE registry_tasks ADD COLUMN registry_task_created_by BIGINT DEFAULT 0;

CREATE INDEX idx_registry_tasks_created_by ON registry_tasks (registry_task_created_by);
CREATE INDEX idx_registry_tasks_kind ON registry_tasks (registry_task_kind);
