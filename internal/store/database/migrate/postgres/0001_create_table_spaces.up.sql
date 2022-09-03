CREATE TABLE IF NOT EXISTS spaces (
 space_id           SERIAL PRIMARY KEY
,space_name         TEXT
,space_fqsn         TEXT
,space_parentId     INTEGER
,space_description  TEXT
,space_created      INTEGER
,space_updated      INTEGER
,UNIQUE(space_fqsn)
);
