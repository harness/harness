CREATE TABLE spaces (
 space_id           SERIAL PRIMARY KEY
,space_parent_id    INTEGER
,space_uid          TEXT
,space_description  TEXT
,space_is_public    BOOLEAN
,space_created_by   INTEGER
,space_created      BIGINT
,space_updated      BIGINT
);