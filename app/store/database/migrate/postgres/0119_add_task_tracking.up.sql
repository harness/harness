CREATE TYPE registry_task_status AS ENUM ('pending', 'processing', 'success', 'failure');

CREATE TABLE registry_tasks (
    registry_task_key         text PRIMARY KEY,
    registry_task_kind        text NOT NULL,
    registry_task_payload     jsonb,
    registry_task_status      registry_task_status NOT NULL DEFAULT 'pending',
    registry_task_run_again   boolean NOT NULL DEFAULT false,
    registry_task_updated_at  BIGINT NOT NULL
);

CREATE TABLE registry_task_sources (
    registry_task_source_key        text NOT NULL REFERENCES registry_tasks(registry_task_key) ON DELETE CASCADE,
    registry_task_source_type   text NOT NULL,
    registry_task_source_id     text NOT NULL,
    registry_task_source_status     registry_task_status NOT NULL DEFAULT 'pending',
    registry_task_source_run_id     text,
    registry_task_source_error      text,
    registry_task_source_updated_at BIGINT NOT NULL,
    PRIMARY KEY (registry_task_source_key, registry_task_source_type, registry_task_source_id)
);

CREATE TABLE registry_task_events (
    registry_task_event_id                 UUID PRIMARY KEY,
    registry_task_event_key                text,
    registry_task_event_event              text,
    registry_task_event_payload            jsonb,
    registry_task_event_created_at         BIGINT NOT NULL
);

CREATE INDEX idx_registry_task_sources_run_id ON registry_task_sources (registry_task_source_run_id);
CREATE INDEX idx_registry_task_sources_status ON registry_task_sources (registry_task_source_status);
CREATE INDEX idx_registry_tasks_status ON registry_tasks (registry_task_status);
