CREATE TABLE principals (
principal_id              SERIAL PRIMARY KEY
,principal_uid            TEXT
,principal_uid_unique     TEXT
,principal_email          TEXT
,principal_type           TEXT
,principal_display_name   TEXT
,principal_admin          BOOLEAN
,principal_blocked        BOOLEAN
,principal_salt           TEXT
,principal_created        BIGINT
,principal_updated        BIGINT

,principal_user_password  TEXT

,principal_sa_parent_type TEXT
,principal_sa_parent_id   INTEGER

,UNIQUE(principal_uid_unique)
);
