ALTER TABLE ai_tasks
    ADD COLUMN aitask_task_id BIGINT,
    ADD COLUMN aitask_model_decision_id BIGINT,
    ADD COLUMN aitask_request_budget INTEGER NOT NULL DEFAULT 60,
    ADD COLUMN aitask_requests_used INTEGER NOT NULL DEFAULT 0;

ALTER TABLE ai_tasks
    ADD CONSTRAINT fk_ai_tasks_task_id FOREIGN KEY (aitask_task_id)
        REFERENCES tasks (task_id) ON DELETE SET NULL;

CREATE INDEX ai_tasks_task_id ON ai_tasks(aitask_task_id);
