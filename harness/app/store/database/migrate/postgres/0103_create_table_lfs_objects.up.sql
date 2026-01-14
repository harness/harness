CREATE TABLE lfs_objects (
     lfs_object_id SERIAL PRIMARY KEY
    ,lfs_object_oid TEXT NOT NULL
    ,lfs_object_size BIGINT NOT NULL
    ,lfs_object_created BIGINT NOT NULL
    ,lfs_object_created_by INTEGER NOT NULL
    ,lfs_object_repo_id INTEGER
    ,CONSTRAINT fk_lfs_object_repo_id FOREIGN KEY (lfs_object_repo_id)
        REFERENCES repositories (repo_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE SET NULL
    ,CONSTRAINT fk_lfs_object_created_by FOREIGN KEY (lfs_object_created_by)
        REFERENCES principals (principal_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

CREATE UNIQUE INDEX lfs_objects_oid
    ON lfs_objects(lfs_object_repo_id, lfs_object_oid);