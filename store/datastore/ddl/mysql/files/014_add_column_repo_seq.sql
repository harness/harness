-- name: alter-table-add-repo-seq

ALTER TABLE repos ADD COLUMN repo_counter INTEGER;

-- name: update-table-set-repo-seq

UPDATE repos SET repo_counter = (
  SELECT max(build_number)
  FROM builds
  WHERE builds.build_repo_id = repos.repo_id
)

-- name: update-table-set-repo-seq-default

UPDATE repos SET repo_counter = 0
WHERE repo_counter IS NULL
