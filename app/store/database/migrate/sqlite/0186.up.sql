ALTER TABLE checks ADD COLUMN check_bypassed_by INTEGER REFERENCES principals(principal_id) ON DELETE SET NULL;
