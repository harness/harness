CREATE TABLE IF NOT EXISTS spaces (
 space_id           SERIAL PRIMARY KEY
,space_name         TEXT
,space_fqn         TEXT
,space_parentId     INTEGER
,space_displayName  TEXT
,space_description  TEXT
,space_isPublic     BOOLEAN
,space_createdBy    INTEGER
,space_created      INTEGER
,space_updated      INTEGER
,UNIQUE(space_fqn)
);