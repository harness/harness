CREATE TABLE ai_models (
     aimodel_id                    TEXT PRIMARY KEY
    ,aimodel_kind                  TEXT NOT NULL
    ,aimodel_context_length        INTEGER NOT NULL DEFAULT 0
    ,aimodel_supports_tools        BOOLEAN NOT NULL DEFAULT FALSE
    ,aimodel_status                TEXT NOT NULL DEFAULT 'active'
    ,aimodel_success_rate_by_task  TEXT NOT NULL DEFAULT '{}'
    ,aimodel_p95_latency_ms        INTEGER NOT NULL DEFAULT 0
    ,aimodel_last_seen             BIGINT NOT NULL DEFAULT 0
    ,aimodel_quarantined_reason    TEXT NOT NULL DEFAULT ''
    ,aimodel_created               BIGINT NOT NULL
    ,aimodel_updated               BIGINT NOT NULL
);

-- aidecision_task_id intentionally has no foreign key; the tasks table is created in a later migration.
CREATE TABLE ai_decisions (
     aidecision_id          INTEGER PRIMARY KEY AUTOINCREMENT
    ,aidecision_task_id     BIGINT
    ,aidecision_question_id TEXT NOT NULL
    ,aidecision_model_build TEXT NOT NULL
    ,aidecision_state_hash  TEXT NOT NULL
    ,aidecision_answer      TEXT NOT NULL DEFAULT '{}'
    ,aidecision_fallback    BOOLEAN NOT NULL DEFAULT FALSE
    ,aidecision_outcome     TEXT
    ,aidecision_created     BIGINT NOT NULL
);

CREATE INDEX ai_decisions_task_id ON ai_decisions(aidecision_task_id);
CREATE INDEX ai_decisions_state_hash ON ai_decisions(aidecision_state_hash);

CREATE TABLE ai_quota (
     aiquota_day      TEXT NOT NULL
    ,aiquota_class    TEXT NOT NULL
    ,aiquota_used     INTEGER NOT NULL DEFAULT 0
    ,aiquota_reserved INTEGER NOT NULL DEFAULT 0
    ,PRIMARY KEY (aiquota_day, aiquota_class)
);

CREATE TABLE ai_usage (
     aiusage_id         INTEGER PRIMARY KEY AUTOINCREMENT
    ,aiusage_space_id   BIGINT
    ,aiusage_repo_id    BIGINT
    ,aiusage_feature    TEXT NOT NULL
    ,aiusage_model      TEXT NOT NULL
    ,aiusage_requests   INTEGER NOT NULL DEFAULT 0
    ,aiusage_tokens_in  BIGINT NOT NULL DEFAULT 0
    ,aiusage_tokens_out BIGINT NOT NULL DEFAULT 0
    ,aiusage_cost_usd   REAL NOT NULL DEFAULT 0
    ,aiusage_created    BIGINT NOT NULL
);

CREATE INDEX ai_usage_created ON ai_usage(aiusage_created);
