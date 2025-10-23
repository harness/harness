create index if not exists  idx_registries_parent_id_package_type
    ON registries (registry_parent_id, registry_package_type);

create index if not exists  idx_tags_registry_id_image_name_updated_at
    ON tags (tag_registry_id, tag_image_name, tag_updated_at);
