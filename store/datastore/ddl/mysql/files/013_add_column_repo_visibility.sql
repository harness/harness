-- name: alter-table-add-repo-visibility

ALTER TABLE repos ADD COLUMN repo_visibility INTEGER

-- name: update-table-set-repo-visibility

UPDATE repos
SET repo_visibility = CASE
  WHEN repo_private = 0 THEN 'public'
  ELSE 'private'
  END
