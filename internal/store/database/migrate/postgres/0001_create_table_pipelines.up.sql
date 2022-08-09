CREATE TABLE IF NOT EXISTS pipelines (
 pipeline_id          SERIAL PRIMARY KEY
,pipeline_name        TEXT
,pipeline_slug        TEXT
,pipeline_desc        TEXT
,pipeline_token       TEXT
,pipeline_active      BOOLEAN
,pipeline_created     INTEGER
,pipeline_updated     INTEGER
,UNIQUE(pipeline_token)
,UNIQUE(pipeline_slug)
);
