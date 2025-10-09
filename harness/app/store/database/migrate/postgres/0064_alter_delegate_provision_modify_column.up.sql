ALTER TABLE delegate_provision_details
    DROP CONSTRAINT fk_dpdeta_gitspace_instance_id,
    DROP COLUMN dpdeta_gitspace_instance_id,
    ADD COLUMN dpdeta_gitspace_instance_identifier TEXT NOT NULL,
    ADD CONSTRAINT fk_dpdeta_gitspace_instance_identifier_space_id FOREIGN KEY (dpdeta_gitspace_instance_identifier, dpdeta_space_id)
        REFERENCES gitspaces (gits_uid, gits_space_id) MATCH SIMPLE
        ON UPDATE NO ACTION;