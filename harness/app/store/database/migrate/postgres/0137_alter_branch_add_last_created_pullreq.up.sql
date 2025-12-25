ALTER TABLE branches
ADD COLUMN branch_last_created_pullreq_id INTEGER DEFAULT NULL;

ALTER TABLE branches
ADD CONSTRAINT fk_branches_last_created_pullreq_id
FOREIGN KEY (branch_last_created_pullreq_id)
REFERENCES pullreqs (pullreq_id) MATCH SIMPLE
ON UPDATE NO ACTION
ON DELETE SET NULL;
