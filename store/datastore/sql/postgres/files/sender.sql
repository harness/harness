-- name: sender-find-repo

SELECT
 sender_id
,sender_repo_id
,sender_login
,sender_allow
,sender_block
FROM senders
WHERE sender_repo_id = $1

-- name: sender-find-repo-login

SELECT
 sender_id
,sender_repo_id
,sender_login
,sender_allow
,sender_block
FROM senders
WHERE sender_repo_id = $1
  AND sender_login = $2

-- name: sender-delete-repo

DELETE FROM senders WHERE sender_repo_id = $1

-- name: sender-delete

DELETE FROM senders WHERE sender_id = $1
