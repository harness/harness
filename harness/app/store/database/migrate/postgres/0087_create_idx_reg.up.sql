ALTER TABLE registries ADD CONSTRAINT unique_registry_parent_name UNIQUE (registry_parent_id, registry_name);
CREATE INDEX idx_spaces_lower_space_uid ON spaces (LOWER(space_uid));
