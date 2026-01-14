create index if not exists index_manifest_registry_id_manifest_image_name_manifest_created_at_idx
    on manifests (manifest_registry_id, manifest_image_name, manifest_created_at DESC);