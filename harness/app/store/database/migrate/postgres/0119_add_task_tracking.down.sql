-- Drop tables and types in reverse order of dependencies
DROP TABLE IF EXISTS registry_task_events;
DROP TABLE IF EXISTS registry_task_sources;
DROP TABLE IF EXISTS registry_tasks;
DROP TYPE IF EXISTS registry_task_status;
