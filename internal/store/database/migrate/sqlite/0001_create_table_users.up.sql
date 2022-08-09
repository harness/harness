CREATE TABLE IF NOT EXISTS users (
 user_id            INTEGER PRIMARY KEY AUTOINCREMENT
,user_email         TEXT COLLATE NOCASE
,user_password      TEXT
,user_salt          TEXT
,user_name          TEXT
,user_company       TEXT
,user_admin         BOOLEAN
,user_blocked       BOOLEAN
,user_created       INTEGER
,user_updated       INTEGER
,user_authed        INTEGER
,UNIQUE(user_salt)
,UNIQUE(user_email COLLATE NOCASE)
);
