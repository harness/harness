CREATE TABLE IF NOT EXISTS pipelines (
 pipeline_id          INTEGER PRIMARY KEY AUTOINCREMENT
,pipeline_name        TEXT
,pipeline_slug        TEXT COLLATE NOCASE
,pipeline_desc        TEXT
,pipeline_token       TEXT
,pipeline_active      BOOLEAN
,pipeline_created     INTEGER
,pipeline_updated     INTEGER
,UNIQUE(pipeline_token)
,UNIQUE(pipeline_slug COLLATE NOCASE)
);