-- primary key is required by some database tools and dependencies.
-- 1. Create a new table with primary key
CREATE TABLE migrations_with_primary_key (
  version TEXT PRIMARY KEY
);

-- 2. Copy data (without this step, the new migration table would be empty after startup on a fresh db)
INSERT INTO migrations_with_primary_key(version)
SELECT version
FROM migrations;

-- 3. Drop the old table
DROP TABLE migrations;

-- 4. Rename the new table
ALTER TABLE migrations_with_primary_key RENAME TO migrations;
