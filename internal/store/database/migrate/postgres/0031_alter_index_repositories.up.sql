DROP INDEX repositories_parent_id;
CREATE UNIQUE INDEX repositories_parent_id_uid ON repositories(repo_parent_id, LOWER(repo_uid));