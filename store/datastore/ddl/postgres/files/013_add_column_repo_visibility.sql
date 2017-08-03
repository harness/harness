-- name: alter-table-add-repo-visibility

ALTER TABLE repos ADD COLUMN repo_visibility VARCHAR(50)

-- name: update-table-set-repo-visibility

UPDATE repos
SET repo_visibility = (CASE
  WHEN repo_private = false THEN 'public'
  ELSE 'private'
  END)
