ALTER TABLE branches
DROP CONSTRAINT IF EXISTS fk_branches_last_created_pullreq_id;

ALTER TABLE branches
DROP COLUMN IF EXISTS branch_last_created_pullreq_id;
