CREATE TABLE IF NOT EXISTS executions (
 execution_id           INTEGER PRIMARY KEY AUTOINCREMENT
,execution_pipeline_id  INTEGER
,execution_slug         TEXT COLLATE NOCASE
,execution_name         TEXT
,execution_desc         TEXT
,execution_created      INTEGER
,execution_updated      INTEGER
,UNIQUE(execution_pipeline_id, execution_slug)
);
