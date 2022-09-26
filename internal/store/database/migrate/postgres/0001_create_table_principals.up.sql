CREATE TABLE IF NOT EXISTS principals (
principal_id              SERIAL PRIMARY KEY
,principal_type           TEXT
,principal_name           TEXT
,principal_admin          BOOLEAN
,principal_externalId     TEXT
,principal_blocked        BOOLEAN
,principal_salt           TEXT
,principal_created        INTEGER
,principal_updated        INTEGER

,principal_user_email     CITEXT
,principal_user_password  TEXT

,principal_sa_parentType  TEXT
,principal_sa_parentId    INTEGER

,UNIQUE(principal_salt)
,UNIQUE(principal_user_email)
);
