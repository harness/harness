-- At most one active task per (repo, origin, origin_ref) for non manual origins.
-- Closes the race between the lookup and the insert of the deduplication of the task service.
CREATE UNIQUE INDEX tasks_repo_id_origin_origin_ref_active
    ON tasks (task_repo_id, task_origin, task_origin_ref)
    WHERE task_origin <> 'manual' AND task_state NOT IN ('descartada', 'verificada');
