CREATE TABLE branches_new (
    branch_repo_id INTEGER NOT NULL,
    branch_name TEXT NOT NULL,
    branch_sha TEXT NOT NULL,
    branch_created_by INTEGER NOT NULL,
    branch_created BIGINT NOT NULL,
    branch_updated_by INTEGER NOT NULL,
    branch_updated BIGINT NOT NULL,
    branch_last_created_pullreq_id INTEGER DEFAULT NULL,
    
    PRIMARY KEY (branch_repo_id, branch_name),
    
    FOREIGN KEY (branch_repo_id)
        REFERENCES repositories (repo_id)
        ON DELETE CASCADE,
    
    FOREIGN KEY (branch_created_by)
        REFERENCES principals (principal_id)
        ON DELETE NO ACTION,
    
    FOREIGN KEY (branch_updated_by)
        REFERENCES principals (principal_id)
        ON DELETE NO ACTION,
    
    FOREIGN KEY (branch_last_created_pullreq_id)
        REFERENCES pullreqs (pullreq_id)
        ON DELETE SET NULL
);

INSERT INTO branches_new
SELECT 
    branch_repo_id,
    branch_name,
    branch_sha,
    branch_created_by,
    branch_created,
    branch_updated_by,
    branch_updated,
    NULL AS branch_last_created_pullreq_id
FROM branches;

DROP TABLE branches;

ALTER TABLE branches_new RENAME TO branches;

CREATE INDEX idx_branches_updated_by_updated ON branches (branch_updated_by, branch_updated);