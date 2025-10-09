ALTER TABLE spaces DROP COLUMN space_deleted;

DROP INDEX spaces_parent_id;
DROP INDEX spaces_deleted;

CREATE INDEX spaces_parent_id
ON spaces(space_parent_id);