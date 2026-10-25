DROP INDEX IF EXISTS ai_tasks_task_id;

ALTER TABLE ai_tasks DROP COLUMN aitask_requests_used;
ALTER TABLE ai_tasks DROP COLUMN aitask_request_budget;
ALTER TABLE ai_tasks DROP COLUMN aitask_model_decision_id;
ALTER TABLE ai_tasks DROP COLUMN aitask_task_id;
