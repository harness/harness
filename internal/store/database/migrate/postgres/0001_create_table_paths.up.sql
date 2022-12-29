CREATE TABLE paths (
 path_id           SERIAL PRIMARY KEY
,path_value        TEXT
,path_value_unique TEXT
,path_is_alias     BOOLEAN
,path_target_type  TEXT
,path_target_id    INTEGER
,path_created_by   INTEGER
,path_created      BIGINT
,path_updated      BIGINT
,UNIQUE(path_value_unique)
);