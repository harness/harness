-- name: repo-update-counter

UPDATE repos SET repo_counter = ?
WHERE repo_counter = ?
  AND repo_id = ?
