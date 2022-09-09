CREATE TABLE IF NOT EXISTS spaces (
 space_id           INTEGER PRIMARY KEY AUTOINCREMENT
,space_name         TEXT COLLATE NOCASE
,space_parentId     INTEGER
,space_displayName  TEXT
,space_description  TEXT
,space_isPublic     BOOLEAN
,space_createdBy    INTEGER
,space_created      INTEGER
,space_updated      INTEGER
);