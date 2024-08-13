ALTER TABLE delegate_provision_details
    DROP CONSTRAINT fk_dpdeta_gitspace_instance_identifier_space_id,
    DROP COLUMN dpdeta_gitspace_instance_identifier,
    ADD COLUMN dpdeta_gitspace_instance_id INTEGER NOT NULL,
    ADD CONSTRAINT fk_dpdeta_gitspace_instance_id FOREIGN KEY (dpdeta_gitspace_instance_id)
        REFERENCES gitspaces (gits_id) MATCH SIMPLE
        ON UPDATE NO ACTION;