ALTER TABLE repositories DROP COLUMN repo_deleted;

DROP INDEX repositories_parent_id_uid;
DROP INDEX repositories_deleted;

CREATE UNIQUE INDEX repositories_parent_id_uid
ON repositories(repo_parent_id, LOWER(repo_uid));
