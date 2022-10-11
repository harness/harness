CREATE TABLE IF NOT EXISTS paths (
 path_id          INTEGER PRIMARY KEY AUTOINCREMENT
,path_value       TEXT COLLATE NOCASE
,path_isAlias     BOOLEAN
,path_targetType  TEXT
,path_targetId    INTEGER
,path_createdBy   INTEGER
,path_created     BIGINT
,path_updated     BIGINT
,UNIQUE(path_value COLLATE NOCASE)
);