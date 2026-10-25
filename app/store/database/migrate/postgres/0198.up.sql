CREATE TABLE agent_attempts (
     agent_attempt_id       BIGSERIAL PRIMARY KEY
    ,agent_attempt_repo_id  BIGINT NOT NULL
    ,agent_attempt_task_key TEXT NOT NULL
    ,agent_attempt_model    TEXT NOT NULL DEFAULT ''
    ,agent_attempt_state    TEXT NOT NULL
    ,agent_attempt_started  BIGINT NOT NULL
    ,agent_attempt_finished BIGINT
    ,agent_attempt_error    TEXT NOT NULL DEFAULT ''
);

CREATE INDEX agent_attempts_repo_id_state ON agent_attempts(agent_attempt_repo_id, agent_attempt_state);
CREATE INDEX agent_attempts_task_key_started ON agent_attempts(agent_attempt_task_key, agent_attempt_started);
