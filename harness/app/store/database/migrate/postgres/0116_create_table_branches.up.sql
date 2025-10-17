CREATE TABLE branches (
 branch_repo_id           INTEGER NOT NULL,
 branch_name              TEXT NOT NULL,
 branch_sha               TEXT NOT NULL,
 branch_created_by        INTEGER NOT NULL,
 branch_created           BIGINT NOT NULL,
 branch_updated_by        INTEGER NOT NULL,
 branch_updated           BIGINT NOT NULL,

 -- Define primary key first for better readability
 PRIMARY KEY (branch_repo_id, branch_name),

 CONSTRAINT fk_branch_repo_id FOREIGN KEY (branch_repo_id)
    REFERENCES repositories (repo_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE,

 CONSTRAINT fk_branch_created_by FOREIGN KEY (branch_created_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION,

 CONSTRAINT fk_branch_updated_by FOREIGN KEY (branch_updated_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
);

-- Index on updated_by and updated columns for filtering recent branches by user
CREATE INDEX idx_branches_updated_by_updated ON branches (branch_updated_by, branch_updated);
