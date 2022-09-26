CREATE TABLE IF NOT EXISTS paths (
 path_id          SERIAL PRIMARY KEY
,path_value       TEXT
,path_isAlias     BOOLEAN
,path_targetType  TEXT
,path_targetId    INTEGER
,path_createdBy   INTEGER
,path_created     INTEGER
,path_updated     INTEGER
,UNIQUE(path_value)
);