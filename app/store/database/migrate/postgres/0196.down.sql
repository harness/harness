DROP INDEX IF EXISTS ai_tasks_task_id;

ALTER TABLE ai_tasks
    DROP CONSTRAINT IF EXISTS fk_ai_tasks_task_id;

ALTER TABLE ai_tasks
    DROP COLUMN IF EXISTS aitask_requests_used,
    DROP COLUMN IF EXISTS aitask_request_budget,
    DROP COLUMN IF EXISTS aitask_model_decision_id,
    DROP COLUMN IF EXISTS aitask_task_id;
