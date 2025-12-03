-- Step 1: Create a New Table with the `UNIQUE` Constraint, using `bigint` exactly as in the original schema
CREATE TABLE registries_new (
    registry_id               INTEGER PRIMARY KEY AUTOINCREMENT,
    registry_name             text not null
        CONSTRAINT registry_name_len_check
            CHECK (length(registry_name) <= 255),
    registry_root_parent_id   INTEGER not null,
    registry_parent_id        INTEGER not null,
    registry_description      text,
    registry_type             text not null,
    registry_package_type     text not null,
    registry_upstream_proxies text,
    registry_allowed_pattern  text,
    registry_blocked_pattern  text,
    registry_labels           text,
    registry_created_at       BIGINT not null,
    registry_updated_at       BIGINT not null,
    registry_created_by       INTEGER not null,
    registry_updated_by       INTEGER not null,
    CONSTRAINT unique_registries
        UNIQUE (registry_root_parent_id, registry_name),
    CONSTRAINT unique_registry_parent_name
        UNIQUE (registry_parent_id, registry_name)
);

-- Step 2: Copy Data from the Old Table, specifying columns explicitly to ensure proper mapping
INSERT INTO registries_new (
    registry_id,
    registry_name,
    registry_root_parent_id,
    registry_parent_id,
    registry_description,
    registry_type,
    registry_package_type,
    registry_upstream_proxies,
    registry_allowed_pattern,
    registry_blocked_pattern,
    registry_labels,
    registry_created_at,
    registry_updated_at,
    registry_created_by,
    registry_updated_by
)
SELECT
    registry_id,
    registry_name,
    registry_root_parent_id,
    registry_parent_id,
    registry_description,
    registry_type,
    registry_package_type,
    registry_upstream_proxies,
    registry_allowed_pattern,
    registry_blocked_pattern,
    registry_labels,
    registry_created_at,
    registry_updated_at,
    registry_created_by,
    registry_updated_by
FROM registries;

-- Step 3: Drop the Old Table and Rename the New Table
DROP TABLE registries;
ALTER TABLE registries_new RENAME TO registries;

CREATE INDEX idx_spaces_lower_space_uid ON spaces (LOWER(space_uid));
