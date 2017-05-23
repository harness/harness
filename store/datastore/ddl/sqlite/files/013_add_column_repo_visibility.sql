-- name: alter-table-add-repo-visibility

ALTER TABLE repos ADD COLUMN repo_visibility TEXT

-- name: update-table-set-repo-visibility

UPDATE repos
SET repo_visibility = CASE
  WHEN repo_private = 0 THEN 'public'
  ELSE 'private'
  END
