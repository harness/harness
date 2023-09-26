CREATE TABLE paths (
 path_id            SERIAL PRIMARY KEY
,path_version       INTEGER NOT NULL DEFAULT 0
,path_value         TEXT NOT NULL
,path_value_unique  TEXT NOT NULL
,path_is_primary    BOOLEAN DEFAULT NULL
,path_repo_id       INTEGER
,path_space_id      INTEGER
,path_created_by    INTEGER NOT NULL
,path_created       BIGINT NOT NULL
,path_updated       BIGINT NOT NULL
,UNIQUE(path_value_unique)

,CONSTRAINT fk_path_created_by FOREIGN KEY (path_created_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
,CONSTRAINT fk_path_space_id FOREIGN KEY (path_space_id)
    REFERENCES spaces (space_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_path_repo_id FOREIGN KEY (path_repo_id)
    REFERENCES repositories (repo_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
);