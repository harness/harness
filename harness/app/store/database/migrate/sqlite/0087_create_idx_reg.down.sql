-- Step 1: Create a New Table Without the `UNIQUE` Constraint
CREATE TABLE registries_old (
  registry_root_parent_id INTEGER NOT NULL,
  registry_parent_id INTEGER NOT NULL,
  registry_created_at BIGINT NOT NULL,
  registry_updated_at BIGINT NOT NULL,
  registry_created_by INTEGER NOT NULL,
  registry_updated_by INTEGER NOT NULL,
  registry_id INTEGER PRIMARY KEY,
  registry_blocked_pattern TEXT,
  registry_labels TEXT,
  registry_name TEXT NOT NULL,
  registry_description TEXT,
  registry_type TEXT NOT NULL,
  registry_package_type TEXT NOT NULL,
  registry_upstream_proxies TEXT,
  registry_allowed_pattern TEXT
);

-- Step 2: Copy Data from the Current Table to the New Table
INSERT INTO registries_old
SELECT *
FROM registries;

-- Step 3: Drop the Table with the Constraint and Rename the New Table
DROP TABLE registries;
ALTER TABLE registries_old RENAME TO registries;

DROP INDEX idx_spaces_lower_space_uid;
