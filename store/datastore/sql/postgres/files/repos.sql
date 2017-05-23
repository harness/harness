-- name: repo-update-counter

UPDATE repos SET repo_counter = $1
WHERE repo_counter = $2
  AND repo_id = $3
