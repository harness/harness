-- name: repo-update-counter

UPDATE repos SET repo_counter = ?
WHERE repo_counter = ?
  AND repo_id = ?

-- name: repo-find-user

SELECT
 repo_id
,repo_user_id
,repo_owner
,repo_name
,repo_full_name
,repo_avatar
,repo_link
,repo_clone
,repo_branch
,repo_timeout
,repo_private
,repo_trusted
,repo_active
,repo_allow_pr
,repo_allow_push
,repo_allow_deploys
,repo_allow_tags
,repo_hash
,repo_scm
,repo_config_path
,repo_gated
,repo_visibility
,repo_counter
FROM repos
INNER JOIN perms ON perms.perm_repo_id = repos.repo_id
WHERE perms.perm_user_id = ?
ORDER BY repo_full_name ASC

-- name: repo-insert-ignore

INSERT IGNORE INTO repos (
 repo_user_id
,repo_owner
,repo_name
,repo_full_name
,repo_avatar
,repo_link
,repo_clone
,repo_branch
,repo_timeout
,repo_private
,repo_trusted
,repo_active
,repo_allow_pr
,repo_allow_push
,repo_allow_deploys
,repo_allow_tags
,repo_hash
,repo_scm
,repo_config_path
,repo_gated
,repo_visibility
,repo_counter
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)

-- name: repo-delete

DELETE FROM repos WHERE repo_id = ?
