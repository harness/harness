CREATE TABLE IF NOT EXISTS spaces (
 space_id           INTEGER PRIMARY KEY AUTOINCREMENT
,space_name         TEXT
,space_fqsn         TEXT COLLATE NOCASE
,space_parentId     INTEGER
,space_description  TEXT
,space_created      INTEGER
,space_updated      INTEGER
,UNIQUE(space_fqsn COLLATE NOCASE)
);