ALTER TABLE repositories ADD COLUMN repo_deleted BIGINT DEFAULT NULL;

DROP INDEX repositories_parent_id_uid;

CREATE UNIQUE INDEX repositories_parent_id_uid
    ON repositories(repo_parent_id, LOWER(repo_uid))
    WHERE repo_deleted IS NULL;

CREATE INDEX repositories_deleted
    ON repositories(repo_deleted)
    WHERE repo_deleted IS NOT NULL;