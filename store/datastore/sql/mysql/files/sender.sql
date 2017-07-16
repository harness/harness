-- name: sender-find-repo

SELECT
 sender_id
,sender_repo_id
,sender_login
,sender_allow
,sender_block
FROM senders
WHERE sender_repo_id = ?

-- name: sender-find-repo-login

SELECT
 sender_id
,sender_repo_id
,sender_login
,sender_allow
,sender_block
FROM senders
WHERE sender_repo_id = ?
  AND sender_login = ?

-- name: sender-delete-repo

DELETE FROM senders WHERE sender_repo_id = ?

-- name: sender-delete

DELETE FROM senders WHERE sender_id = ?
