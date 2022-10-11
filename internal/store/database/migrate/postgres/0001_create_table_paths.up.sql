CREATE TABLE IF NOT EXISTS paths (
 path_id          SERIAL PRIMARY KEY
,path_value       TEXT
,path_isAlias     BOOLEAN
,path_targetType  TEXT
,path_targetId    INTEGER
,path_createdBy   INTEGER
,path_created     BIGINT
,path_updated     BIGINT
,UNIQUE(path_value)
);