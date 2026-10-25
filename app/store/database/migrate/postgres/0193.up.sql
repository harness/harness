-- Per-repository task number counter (same idea as repo_pullreq_seq).
ALTER TABLE repositories ADD COLUMN repo_task_seq BIGINT NOT NULL DEFAULT 0;

CREATE TABLE tasks (
    task_id                    SERIAL PRIMARY KEY,
    task_repo_id               INTEGER NOT NULL,
    task_number                BIGINT NOT NULL,
    task_title                 TEXT NOT NULL,
    task_description           TEXT NOT NULL DEFAULT '',
    task_origin                TEXT NOT NULL,
    task_origin_ref            TEXT NOT NULL DEFAULT '',
    task_priority              TEXT NOT NULL,
    task_state                 TEXT NOT NULL,
    task_state_reason          TEXT NOT NULL DEFAULT '',
    task_assignee_principal_id INTEGER DEFAULT NULL,
    task_assignee_group_id     INTEGER DEFAULT NULL,
    task_assignee_is_agent     BOOLEAN NOT NULL DEFAULT FALSE,
    task_due                   BIGINT DEFAULT NULL,
    task_parent_id             INTEGER DEFAULT NULL,
    task_created_by            INTEGER NOT NULL,
    task_created               BIGINT NOT NULL,
    task_updated               BIGINT NOT NULL,
    task_resolved              BIGINT DEFAULT NULL,
    task_verified              BIGINT DEFAULT NULL,
    task_version               BIGINT NOT NULL DEFAULT 0,

    CONSTRAINT fk_tasks_repo_id FOREIGN KEY (task_repo_id)
        REFERENCES repositories (repo_id) ON DELETE CASCADE,
    CONSTRAINT fk_tasks_assignee_principal_id FOREIGN KEY (task_assignee_principal_id)
        REFERENCES principals (principal_id) ON DELETE SET NULL,
    CONSTRAINT fk_tasks_assignee_group_id FOREIGN KEY (task_assignee_group_id)
        REFERENCES usergroups (usergroup_id) ON DELETE SET NULL,
    CONSTRAINT fk_tasks_parent_id FOREIGN KEY (task_parent_id)
        REFERENCES tasks (task_id) ON DELETE SET NULL,
    CONSTRAINT fk_tasks_created_by FOREIGN KEY (task_created_by)
        REFERENCES principals (principal_id)
);

CREATE UNIQUE INDEX tasks_repo_id_number ON tasks (task_repo_id, task_number);
CREATE INDEX tasks_repo_id_state ON tasks (task_repo_id, task_state);
CREATE INDEX tasks_repo_id_priority ON tasks (task_repo_id, task_priority);
CREATE INDEX tasks_assignee_principal_id ON tasks (task_assignee_principal_id);
CREATE INDEX tasks_assignee_group_id ON tasks (task_assignee_group_id);
CREATE INDEX tasks_parent_id ON tasks (task_parent_id);
CREATE INDEX tasks_origin_ref ON tasks (task_origin, task_origin_ref);

CREATE TABLE task_activities (
    task_activity_id           SERIAL PRIMARY KEY,
    task_activity_task_id      INTEGER NOT NULL,
    task_activity_kind         TEXT NOT NULL,
    task_activity_payload      TEXT NOT NULL DEFAULT '{}',
    task_activity_principal_id INTEGER DEFAULT NULL,
    task_activity_created      BIGINT NOT NULL,

    CONSTRAINT fk_task_activities_task_id FOREIGN KEY (task_activity_task_id)
        REFERENCES tasks (task_id) ON DELETE CASCADE,
    CONSTRAINT fk_task_activities_principal_id FOREIGN KEY (task_activity_principal_id)
        REFERENCES principals (principal_id) ON DELETE SET NULL
);

CREATE INDEX task_activities_task_id_created ON task_activities (task_activity_task_id, task_activity_created);

CREATE TABLE task_labels (
    task_label_task_id        INTEGER NOT NULL,
    task_label_label_id       INTEGER NOT NULL,
    task_label_label_value_id INTEGER DEFAULT NULL,
    task_label_created        BIGINT NOT NULL,
    task_label_updated        BIGINT NOT NULL,
    task_label_created_by     INTEGER NOT NULL,
    task_label_updated_by     INTEGER NOT NULL,

    CONSTRAINT fk_task_labels_task_id FOREIGN KEY (task_label_task_id)
        REFERENCES tasks (task_id) ON DELETE CASCADE,
    CONSTRAINT fk_task_labels_label_id FOREIGN KEY (task_label_label_id)
        REFERENCES labels (label_id) ON DELETE CASCADE,
    CONSTRAINT fk_task_labels_label_value_id FOREIGN KEY (task_label_label_value_id)
        REFERENCES label_values (label_value_id) ON DELETE SET NULL,
    CONSTRAINT fk_task_labels_created_by FOREIGN KEY (task_label_created_by)
        REFERENCES principals (principal_id),
    CONSTRAINT fk_task_labels_updated_by FOREIGN KEY (task_label_updated_by)
        REFERENCES principals (principal_id),

    PRIMARY KEY (task_label_task_id, task_label_label_id)
);

CREATE TABLE task_links (
    task_link_task_id INTEGER NOT NULL,
    task_link_kind    TEXT NOT NULL,
    task_link_ref_id  TEXT NOT NULL,
    task_link_created BIGINT NOT NULL DEFAULT 0,

    CONSTRAINT fk_task_links_task_id FOREIGN KEY (task_link_task_id)
        REFERENCES tasks (task_id) ON DELETE CASCADE,

    PRIMARY KEY (task_link_task_id, task_link_kind, task_link_ref_id)
);

CREATE INDEX task_links_kind_ref_id ON task_links (task_link_kind, task_link_ref_id);
