-- name: create-table-users

CREATE TABLE IF NOT EXISTS users (
 user_id     INTEGER PRIMARY KEY AUTO_INCREMENT
,user_login  VARCHAR(250)
,user_token  VARCHAR(500)
,user_secret VARCHAR(500)
,user_expiry INTEGER
,user_email  VARCHAR(500)
,user_avatar VARCHAR(500)
,user_active BOOLEAN
,user_admin  BOOLEAN
,user_hash   VARCHAR(500)

,UNIQUE(user_login)
);
