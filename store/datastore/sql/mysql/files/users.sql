-- name: user-find

SELECT
 user_id
,user_login
,user_token
,user_secret
,user_expiry
,user_email
,user_avatar
,user_active
,user_synced
,user_admin
,user_hash
FROM users
ORDER BY user_login ASC

-- name: user-find-login

SELECT
 user_id
,user_login
,user_token
,user_secret
,user_expiry
,user_email
,user_avatar
,user_active
,user_synced
,user_admin
,user_hash
FROM users
WHERE user_login = ?
LIMIT 1

-- name: user-update

UPDATE users
SET
,user_token  = ?
,user_secret = ?
,user_expiry = ?
,user_email  = ?
,user_avatar = ?
,user_active = ?
,user_synced = ?
,user_admin  = ?
,user_hash   = ?
WHERE user_id = ?

-- name: user-delete

DELETE FROM users WHERE user_id = ?
