CREATE TABLE IF NOT EXISTS principals (
principal_id              INTEGER PRIMARY KEY AUTOINCREMENT
,principal_uid            TEXT
,principal_uidUnique      TEXT
,principal_email          TEXT COLLATE NOCASE
,principal_type           TEXT COLLATE NOCASE
,principal_displayName    TEXT
,principal_admin          BOOLEAN
,principal_blocked        BOOLEAN
,principal_salt           TEXT
,principal_created        BIGINT
,principal_updated        BIGINT

,principal_user_password  TEXT

,principal_sa_parentType  TEXT COLLATE NOCASE
,principal_sa_parentId    INTEGER

,UNIQUE(principal_uidUnique)
,UNIQUE(principal_email COLLATE NOCASE)
,UNIQUE(principal_salt)
);
