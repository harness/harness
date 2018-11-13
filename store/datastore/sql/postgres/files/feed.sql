-- name: feed-latest-build

SELECT
 repo_owner
,repo_name
,repo_full_name
,build_number
,build_event
,build_status
,build_created
,build_started
,build_finished
,build_commit
,build_branch
,build_ref
,build_refspec
,build_remote
,build_title
,build_message
,build_author
,build_email
,build_avatar
FROM repos LEFT OUTER JOIN (
	SELECT DISTINCT ON (build_repo_id) * FROM builds
	ORDER BY build_repo_id, build_id DESC
) b ON b.build_repo_id = repos.repo_id
INNER JOIN perms ON perms.perm_repo_id = repos.repo_id
WHERE perms.perm_user_id = $1
  AND repos.repo_active = TRUE
ORDER BY repo_full_name ASC;

-- name: feed

SELECT
 repo_owner
,repo_name
,repo_full_name
,build_number
,build_event
,build_status
,build_created
,build_started
,build_finished
,build_commit
,build_branch
,build_ref
,build_refspec
,build_remote
,build_title
,build_message
,build_author
,build_email
,build_avatar
FROM repos
INNER JOIN perms  ON perms.perm_repo_id   = repos.repo_id
INNER JOIN builds ON builds.build_repo_id = repos.repo_id
WHERE perms.perm_user_id = $1
ORDER BY build_id DESC
LIMIT 50
