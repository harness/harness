ALTER TABLE spaces ADD COLUMN space_deleted BIGINT DEFAULT NULL;

DROP INDEX spaces_parent_id;

CREATE INDEX spaces_parent_id
    ON spaces(space_parent_id)    
    WHERE space_deleted IS NULL;

CREATE INDEX spaces_deleted_parent_id
    ON spaces(space_deleted, space_parent_id)
    WHERE space_deleted IS NOT NULL;