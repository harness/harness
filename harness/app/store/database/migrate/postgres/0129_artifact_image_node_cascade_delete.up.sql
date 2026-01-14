alter table artifacts
    drop constraint fk_images_image_id;

alter table artifacts
    add constraint fk_images_image_id
        foreign key (artifact_image_id) references images
            on delete cascade;

alter table images
    drop constraint fk_registries_registry_id;

alter table images
    add constraint fk_registries_registry_id
        foreign key (image_registry_id) references registries
            on delete cascade;

alter table nodes
    drop constraint nodes_node_registry_id_fkey;

alter table nodes
    add foreign key (node_registry_id) references registries
        on delete cascade;

create index if not exists layer_manifest_id_idx
    on layers (layer_manifest_id);

create index if not exists manifest_manifest_subject_id_idx
    on manifests (manifest_subject_id);

create index if not exists gc_manifest_id_index
    on gc_manifest_review_queue (manifest_id);

create index if not exists  tag_manifest_id_index
    on tags (tag_manifest_id);

create index if not exists  manifest_ref_child_id_idx
    on manifest_references (manifest_ref_child_id);

create index if not exists  manifest_ref_parent_id_idx
    on manifest_references (manifest_ref_parent_id);

