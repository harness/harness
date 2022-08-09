CREATE TABLE IF NOT EXISTS executions (
 execution_id          SERIAL PRIMARY KEY
,execution_pipeline_id  INTEGER
,execution_slug        TEXT
,execution_name        TEXT
,execution_desc        TEXT
,execution_created     INTEGER
,execution_updated     INTEGER
,UNIQUE(execution_pipeline_id, execution_slug)
);
