CREATE TABLE public_access_registry (
    public_access_registry_id INTEGER PRIMARY KEY,
    CONSTRAINT fk_public_access_registry_id FOREIGN KEY (public_access_registry_id)
        REFERENCES registries (registry_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);