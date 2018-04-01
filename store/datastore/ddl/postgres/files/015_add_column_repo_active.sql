-- name: alter-table-add-repo-active

ALTER TABLE repos ADD COLUMN repo_active BOOLEAN;

-- name: update-table-set-repo-active

UPDATE repos SET repo_active = true;
