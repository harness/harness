CREATE TABLE space_paths (
 space_path_id            SERIAL PRIMARY KEY
,space_path_uid           TEXT NOT NULL
,space_path_uid_unique    TEXT NOT NULL
,space_path_is_primary    BOOLEAN DEFAULT NULL
,space_path_space_id      INTEGER NOT NULL
,space_path_parent_id     INTEGER
,space_path_created_by    INTEGER NOT NULL
,space_path_created       BIGINT NOT NULL
,space_path_updated       BIGINT NOT NULL

,CONSTRAINT fk_space_path_created_by FOREIGN KEY (space_path_created_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
,CONSTRAINT fk_space_path_space_id FOREIGN KEY (space_path_space_id)
    REFERENCES spaces (space_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_space_path_parent_id FOREIGN KEY (space_path_parent_id)
    REFERENCES spaces (space_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
);

CREATE UNIQUE INDEX space_paths_space_id_is_primary
ON space_paths(space_path_space_id, space_path_is_primary);

CREATE UNIQUE INDEX space_paths_uid_unique_no_parent
ON space_paths(space_path_uid_unique)
WHERE space_path_parent_id IS NULL;

CREATE UNIQUE INDEX space_paths_uid_unique
ON space_paths(space_path_parent_id, space_path_uid_unique)
WHERE space_path_parent_id IS NOT NULL;

-- assume no alias paths were created - create fresh primary enries for each space.
INSERT INTO space_paths (
     space_path_uid
    ,space_path_uid_unique
    ,space_path_is_primary
    ,space_path_parent_id
    ,space_path_space_id
    ,space_path_created_by
    ,space_path_created
    ,space_path_updated
) 
SELECT
     space_uid
     -- we assume postgres is used by harness - accountID is case sensitive, rest isn't
    ,CASE WHEN space_parent_id IS NULL
        THEN space_uid
        ELSE LOWER(space_uid)
     END
    ,TRUE
    ,space_parent_id
    ,space_id
    ,space_created_by
    ,space_created
    ,space_updated
FROM spaces;