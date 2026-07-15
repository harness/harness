ALTER TABLE checks
    DROP CONSTRAINT IF EXISTS fk_check_bypassed_by,
    DROP COLUMN IF EXISTS check_bypassed_by;
