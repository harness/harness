-- +migrate Up

CREATE TABLE polls (
 id       SERIAL PRIMARY KEY
,owner    VARCHAR(255)
,name     VARCHAR(255)
,period INTEGER

,UNIQUE(owner, name)
);

INSERT INTO users (user_id, user_login, user_token, user_secret, user_expiry, user_email, user_avatar, user_active, user_admin, user_hash) VALUES (1,'sryadmin','EFDDF4D3-2EB9-400F-BA83-4A9D292A1170','',0,'sryadmin@dataman-inc.net','https://avatars3.githubusercontent.com/u/76609?v=3&s=460',0,1,'sryun-rnd');

-- +migrate Down
DROP TABLE polls;
DELETE FROM users where user_id = 1;
