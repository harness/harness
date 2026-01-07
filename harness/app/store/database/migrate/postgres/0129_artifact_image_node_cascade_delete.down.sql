alter table artifacts
    drop constraint fk_images_image_id;

alter table artifacts
    add constraint fk_images_image_id
        foreign key (artifact_image_id) references images;

alter table images
    drop constraint fk_registries_registry_id;

alter table images
    add constraint fk_registries_registry_id
        foreign key (image_registry_id) references registries;

alter table nodes
    drop constraint nodes_node_registry_id_fkey;

alter table nodes
    add foreign key (node_registry_id) references registries;

drop index if exists layer_manifest_id_idx;
drop index if exists manifest_manifest_subject_id_idx;
drop index if exists gc_manifest_id_index;
drop index if exists tag_manifest_id_index;
drop index if exists manifest_ref_child_id_idx;
drop index if exists manifest_ref_parent_id_idx;