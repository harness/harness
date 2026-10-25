CREATE TABLE insights_projects (
    insights_project_id                    SERIAL PRIMARY KEY,
    insights_project_repo_id               INTEGER NOT NULL,
    insights_project_identifier            TEXT NOT NULL,
    insights_project_public_key            TEXT NOT NULL,
    insights_project_enabled               BOOLEAN NOT NULL DEFAULT TRUE,
    insights_project_rate_limit_per_minute INTEGER NOT NULL DEFAULT 600,
    insights_project_sample_rate           DOUBLE PRECISION NOT NULL DEFAULT 1,
    insights_project_created_by            INTEGER NOT NULL,
    insights_project_created               BIGINT NOT NULL,
    insights_project_updated               BIGINT NOT NULL,

    CONSTRAINT fk_insights_projects_repo_id FOREIGN KEY (insights_project_repo_id)
        REFERENCES repositories (repo_id) ON DELETE CASCADE,
    CONSTRAINT check_insights_projects_sample_rate
        CHECK (insights_project_sample_rate >= 0 AND insights_project_sample_rate <= 1)
);

CREATE UNIQUE INDEX insights_projects_repo_id_identifier
    ON insights_projects (insights_project_repo_id, insights_project_identifier);
CREATE UNIQUE INDEX insights_projects_public_key
    ON insights_projects (insights_project_public_key);

CREATE TABLE insights_issues (
    insights_issue_id                  SERIAL PRIMARY KEY,
    insights_issue_project_id          INTEGER NOT NULL,
    insights_issue_repo_id             INTEGER NOT NULL,
    insights_issue_fingerprint         TEXT NOT NULL,
    insights_issue_title               TEXT NOT NULL DEFAULT '',
    insights_issue_culprit             TEXT NOT NULL DEFAULT '',
    insights_issue_level               TEXT NOT NULL DEFAULT '',
    insights_issue_platform            TEXT NOT NULL DEFAULT '',
    insights_issue_first_seen          BIGINT NOT NULL,
    insights_issue_last_seen           BIGINT NOT NULL,
    insights_issue_times_seen          BIGINT NOT NULL DEFAULT 0,
    insights_issue_users_seen          BIGINT NOT NULL DEFAULT 0,
    insights_issue_last_user           TEXT NOT NULL DEFAULT '',
    insights_issue_first_release       TEXT NOT NULL DEFAULT '',
    insights_issue_last_release        TEXT NOT NULL DEFAULT '',
    insights_issue_status              TEXT NOT NULL,
    insights_issue_resolved_in_release TEXT NOT NULL DEFAULT '',
    insights_issue_task_id             INTEGER DEFAULT NULL,
    insights_issue_created             BIGINT NOT NULL,
    insights_issue_updated             BIGINT NOT NULL,
    insights_issue_version             BIGINT NOT NULL DEFAULT 0,

    CONSTRAINT fk_insights_issues_project_id FOREIGN KEY (insights_issue_project_id)
        REFERENCES insights_projects (insights_project_id) ON DELETE CASCADE,
    CONSTRAINT fk_insights_issues_task_id FOREIGN KEY (insights_issue_task_id)
        REFERENCES tasks (task_id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX insights_issues_project_id_fingerprint
    ON insights_issues (insights_issue_project_id, insights_issue_fingerprint);
CREATE INDEX insights_issues_repo_id_status
    ON insights_issues (insights_issue_repo_id, insights_issue_status);
CREATE INDEX insights_issues_task_id
    ON insights_issues (insights_issue_task_id);

CREATE TABLE insights_events (
    insights_event_id          SERIAL PRIMARY KEY,
    insights_event_issue_id    INTEGER NOT NULL,
    insights_event_project_id  INTEGER NOT NULL,
    insights_event_event_id    TEXT NOT NULL,
    insights_event_timestamp   BIGINT NOT NULL,
    insights_event_release     TEXT NOT NULL DEFAULT '',
    insights_event_environment TEXT NOT NULL DEFAULT '',
    insights_event_level       TEXT NOT NULL DEFAULT '',
    insights_event_message     TEXT NOT NULL DEFAULT '',
    insights_event_payload     TEXT NOT NULL DEFAULT '{}',
    insights_event_created     BIGINT NOT NULL,

    CONSTRAINT fk_insights_events_issue_id FOREIGN KEY (insights_event_issue_id)
        REFERENCES insights_issues (insights_issue_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX insights_events_project_id_event_id
    ON insights_events (insights_event_project_id, insights_event_event_id);
CREATE INDEX insights_events_issue_id_timestamp
    ON insights_events (insights_event_issue_id, insights_event_timestamp);
