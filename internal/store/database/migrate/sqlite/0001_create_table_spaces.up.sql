CREATE TABLE IF NOT EXISTS spaces (
 space_id           INTEGER PRIMARY KEY AUTOINCREMENT
,space_pathName     TEXT COLLATE NOCASE
,space_parentId     INTEGER
,space_name         TEXT
,space_description  TEXT
,space_isPublic     BOOLEAN
,space_createdBy    INTEGER
,space_created      BIGINT
,space_updated      BIGINT
);