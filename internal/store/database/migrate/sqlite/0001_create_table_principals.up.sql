CREATE TABLE IF NOT EXISTS principals (
principal_id              INTEGER PRIMARY KEY AUTOINCREMENT
,principal_uid            TEXT
,principal_type           TEXT
,principal_name           TEXT
,principal_admin          BOOLEAN
,principal_blocked        BOOLEAN
,principal_salt           TEXT
,principal_created        BIGINT
,principal_updated        BIGINT

,principal_user_email     TEXT
,principal_user_password  TEXT

,principal_sa_parentType  TEXT
,principal_sa_parentId    INTEGER

,UNIQUE(principal_uid)
,UNIQUE(principal_salt)
,UNIQUE(principal_user_email COLLATE NOCASE)
);
